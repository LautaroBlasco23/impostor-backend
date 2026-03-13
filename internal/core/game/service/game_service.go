package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/LautaroBlasco23/impostor/internal/core/game/model"
	"github.com/LautaroBlasco23/impostor/internal/core/game/repository"
	roomModel "github.com/LautaroBlasco23/impostor/internal/core/room/model"
	roomRepo "github.com/LautaroBlasco23/impostor/internal/core/room/repository"
	userModel "github.com/LautaroBlasco23/impostor/internal/core/user/model"
	userRepo "github.com/LautaroBlasco23/impostor/internal/core/user/repository"
	wordRepo "github.com/LautaroBlasco23/impostor/internal/core/word/repository"
	"github.com/LautaroBlasco23/impostor/internal/middleware"
	ws "github.com/LautaroBlasco23/impostor/internal/websocket"
	"github.com/google/uuid"
)

const ReconnectTimeout = 60 * time.Second
const LobbyReconnectTimeout = 15 * time.Second

type GameService interface {
	StartGame(ctx context.Context, req *model.StartGameRequest) (*model.Game, error)
	GetGame(ctx context.Context, id string) (*model.Game, error)
	GetGameByRoom(ctx context.Context, roomID string) (*model.Game, error)
	Vote(ctx context.Context, req *model.VoteRequest) (*model.VoteResult, error)
	EndGame(ctx context.Context, gameID string) error
	LeaveGame(ctx context.Context, gameID string, req *model.LeaveGameRequest) error
	CancelGame(ctx context.Context, gameID string, reason string) error
	ReturnToRoom(ctx context.Context, gameID string, req *model.ReturnToRoomRequest) error
	HandleDisconnect(clientID, roomID string)
	HandleReconnect(clientID, roomID, gameID string)
}

type disconnectTimer struct {
	timer    *time.Timer
	gameID   string
	userID   string
	nickname string
}

type gameService struct {
	gameRepo repository.GameRepository
	roomRepo roomRepo.RoomRepository
	userRepo userRepo.UserRepository
	wordRepo wordRepo.WordRepository
	hub      *ws.Hub

	disconnectTimers     map[string]*disconnectTimer
	lobbyDisconnectTimers map[string]*time.Timer
	timersMu             sync.Mutex
}

func NewGameService(
	gameRepo repository.GameRepository,
	roomRepo roomRepo.RoomRepository,
	userRepo userRepo.UserRepository,
	wordRepo wordRepo.WordRepository,
	hub *ws.Hub,
) GameService {
	svc := &gameService{
		gameRepo:              gameRepo,
		roomRepo:              roomRepo,
		userRepo:              userRepo,
		wordRepo:              wordRepo,
		hub:                   hub,
		disconnectTimers:      make(map[string]*disconnectTimer),
		lobbyDisconnectTimers: make(map[string]*time.Timer),
	}

	hub.SetDisconnectHandler(svc.HandleDisconnect)
	hub.SetReconnectHandler(svc.HandleReconnect)
	hub.SetLobbyReconnectHandler(svc.cancelLobbyTimer)

	return svc
}

func (s *gameService) StartGame(ctx context.Context, req *model.StartGameRequest) (*model.Game, error) {
	room, users, err := s.validateAndGetStartGameData(ctx, req.RoomID)
	if err != nil {
		return nil, err
	}

	words, err := s.wordRepo.GetRandomByCategory(ctx, room.Category, middleware.GetLanguage(ctx), 1)
	if err != nil || len(words) == 0 {
		return nil, fmt.Errorf("no words found for category: %s", room.Category)
	}

	impostorID, err := s.assignRoles(ctx, users)
	if err != nil {
		return nil, err
	}

	game := s.createGame(req.RoomID, impostorID, words[0].Text, room.Category)

	if err := s.gameRepo.Create(ctx, game); err != nil {
		return nil, err
	}

	s.notifyGameStart(users, game)

	return game, nil
}

func (s *gameService) validateAndGetStartGameData(ctx context.Context, roomID string) (*roomModel.Room, []*userModel.User, error) {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, nil, fmt.Errorf("room not found: %w", err)
	}

	if !room.IsActive {
		return nil, nil, fmt.Errorf("room is not active")
	}

	if room.Category == "" {
		return nil, nil, fmt.Errorf("room category not set")
	}

	users, err := s.userRepo.GetByRoomID(ctx, roomID)
	if err != nil {
		return nil, nil, err
	}

	if len(users) < 3 {
		return nil, nil, fmt.Errorf("need at least 3 users to start game")
	}

	for _, user := range users {
		if !user.IsReady {
			return nil, nil, fmt.Errorf("all users must be ready")
		}
	}

	return room, users, nil
}

func (s *gameService) assignRoles(ctx context.Context, users []*userModel.User) (string, error) {
	impostorIndex := rand.Intn(len(users))
	impostorID := users[impostorIndex].ID

	for i, user := range users {
		if i == impostorIndex {
			user.Role = userModel.RoleImpostor
		} else {
			user.Role = userModel.RolePlayer
		}
		user.IsAlive = true
		if err := s.userRepo.Update(ctx, user); err != nil {
			return "", err
		}
	}

	return impostorID, nil
}

func (s *gameService) createGame(roomID, impostorID, word, category string) *model.Game {
	return &model.Game{
		ID:                uuid.New().String(),
		RoomID:            roomID,
		State:             model.GameStatePlaying,
		ImpostorID:        impostorID,
		CurrentWord:       word,
		Category:          category,
		VoteCount:         make(map[string]int),
		VotedUsers:        make(map[string]string),
		DisconnectedUsers: make(map[string]*model.DisconnectedUser),
		RoundNumber:       1,
		CreatedAt:         time.Now(),
	}
}

func (s *gameService) notifyGameStart(users []*userModel.User, game *model.Game) {
	for _, user := range users {
		payload := map[string]interface{}{
			"game_id":      game.ID,
			"category":     game.Category,
			"round_number": game.RoundNumber,
			"impostor_id":  game.ImpostorID,
		}

		if user.ID != game.ImpostorID {
			payload["current_word"] = game.CurrentWord
		}

		s.hub.SendToClient(user.ID, game.RoomID, ws.EventGameStarted, payload)
	}
}

func (s *gameService) GetGame(ctx context.Context, id string) (*model.Game, error) {
	return s.gameRepo.GetByID(ctx, id)
}

func (s *gameService) GetGameByRoom(ctx context.Context, roomID string) (*model.Game, error) {
	return s.gameRepo.GetByRoomID(ctx, roomID)
}

func (s *gameService) Vote(ctx context.Context, req *model.VoteRequest) (*model.VoteResult, error) {
	game, err := s.gameRepo.GetByID(ctx, req.GameID)
	if err != nil {
		return nil, err
	}

	s.initializeVoteMaps(game)

	voter, target, err := s.validateVote(ctx, game, req)
	if err != nil {
		return nil, err
	}

	s.recordVote(game, req.VoterID, req.TargetID)
	s.broadcastVote(game.RoomID, voter, target, len(game.VotedUsers))

	aliveUsers, err := s.getAliveUsers(ctx, game.RoomID)
	if err != nil {
		return nil, err
	}

	if len(game.VotedUsers) < len(aliveUsers) {
		return s.handlePartialVoting(ctx, game, aliveUsers)
	}

	return s.handleVotingComplete(ctx, game)
}

func (s *gameService) initializeVoteMaps(game *model.Game) {
	if game.VoteCount == nil {
		game.VoteCount = make(map[string]int)
	}
	if game.VotedUsers == nil {
		game.VotedUsers = make(map[string]string)
	}
}

func (s *gameService) validateVote(ctx context.Context, game *model.Game, req *model.VoteRequest) (*userModel.User, *userModel.User, error) {
	if game.State != model.GameStatePlaying && game.State != model.GameStateVoting {
		return nil, nil, fmt.Errorf("game is not in playing state")
	}

	voter, err := s.userRepo.GetByID(ctx, req.VoterID)
	if err != nil {
		return nil, nil, fmt.Errorf("voter not found: %w", err)
	}

	if !voter.IsAlive {
		return nil, nil, fmt.Errorf("dead users cannot vote")
	}

	if voter.RoomID != game.RoomID {
		return nil, nil, fmt.Errorf("voter is not in the game room")
	}

	target, err := s.userRepo.GetByID(ctx, req.TargetID)
	if err != nil {
		return nil, nil, fmt.Errorf("target not found: %w", err)
	}

	if !target.IsAlive {
		return nil, nil, fmt.Errorf("cannot vote for dead user")
	}

	if _, hasVoted := game.VotedUsers[req.VoterID]; hasVoted {
		return nil, nil, fmt.Errorf("user has already voted this round")
	}

	return voter, target, nil
}

func (s *gameService) recordVote(game *model.Game, voterID, targetID string) {
	game.VotedUsers[voterID] = targetID
	game.VoteCount[targetID]++
	game.State = model.GameStateVoting
}

func (s *gameService) broadcastVote(roomID string, voter, target *userModel.User, votesCast int) {
	s.hub.BroadcastToRoom(roomID, ws.EventUserVoted, map[string]interface{}{
		"voter_id":      voter.ID,
		"voter_name":    voter.Nickname,
		"target_id":     target.ID,
		"target_name":   target.Nickname,
		"votes_cast":    votesCast,
		"total_players": s.countAlivePlayers(roomID),
	})
}

func (s *gameService) handlePartialVoting(ctx context.Context, game *model.Game, aliveUsers []*userModel.User) (*model.VoteResult, error) {
	if err := s.gameRepo.Update(ctx, game); err != nil {
		return nil, err
	}
	return &model.VoteResult{
		GameState: model.GameStateVoting,
		Message:   fmt.Sprintf("Waiting for %d more votes", len(aliveUsers)-len(game.VotedUsers)),
	}, nil
}

func (s *gameService) handleVotingComplete(ctx context.Context, game *model.Game) (*model.VoteResult, error) {
	eliminatedID := s.getMostVotedUser(game.VoteCount)
	eliminatedUser, err := s.userRepo.GetByID(ctx, eliminatedID)
	if err != nil {
		return nil, err
	}

	eliminatedUser.IsAlive = false
	if err := s.userRepo.Update(ctx, eliminatedUser); err != nil {
		return nil, err
	}

	wasImpostor := eliminatedUser.Role == userModel.RoleImpostor

	s.hub.BroadcastToRoom(game.RoomID, ws.EventUserEliminated, map[string]interface{}{
		"user_id":      eliminatedID,
		"nickname":     eliminatedUser.Nickname,
		"was_impostor": wasImpostor,
		"vote_count":   game.VoteCount[eliminatedID],
	})

	var result *model.VoteResult
	if wasImpostor {
		result = s.handleImpostorEliminated(game, eliminatedUser)
	} else {
		var resultErr error
		result, resultErr = s.handlePlayerEliminated(ctx, game, eliminatedUser)
		if resultErr != nil {
			return nil, resultErr
		}
	}

	result.EliminatedUserID = eliminatedID
	result.WasImpostor = wasImpostor

	if updateErr := s.gameRepo.Update(ctx, game); updateErr != nil {
		return nil, updateErr
	}

	return result, nil
}

func (s *gameService) handleImpostorEliminated(game *model.Game, eliminatedUser *userModel.User) *model.VoteResult {
	game.State = model.GameStateWon
	message := fmt.Sprintf("Players win! %s was the impostor!", eliminatedUser.Nickname)

	s.hub.BroadcastToRoom(game.RoomID, ws.EventGameWon, map[string]interface{}{
		"impostor_id":   eliminatedUser.ID,
		"impostor_name": eliminatedUser.Nickname,
		"message":       message,
		"word":          game.CurrentWord,
	})

	return &model.VoteResult{
		GameState: model.GameStateWon,
		Message:   message,
	}
}

func (s *gameService) handlePlayerEliminated(ctx context.Context, game *model.Game, eliminatedUser *userModel.User) (*model.VoteResult, error) {
	aliveAfterVote, err := s.getAliveUsers(ctx, game.RoomID)
	if err != nil {
		return nil, err
	}

	if len(aliveAfterVote) <= 2 {
		return s.handleImpostorWins(game), nil
	}

	return s.handleContinueGame(game, eliminatedUser), nil
}

func (s *gameService) handleImpostorWins(game *model.Game) *model.VoteResult {
	game.State = model.GameStateLost
	message := "Impostor wins! Not enough players remaining!"

	impostorName := "Unknown"
	impostor, err := s.userRepo.GetByID(context.Background(), game.ImpostorID)
	if err != nil {
		log.Printf("impostor not found: %v", err)
	} else if impostor != nil {
		impostorName = impostor.Nickname
	}

	s.hub.BroadcastToRoom(game.RoomID, ws.EventGameLost, map[string]interface{}{
		"impostor_id":   game.ImpostorID,
		"impostor_name": impostorName,
		"message":       message,
		"word":          game.CurrentWord,
	})

	return &model.VoteResult{
		GameState: model.GameStateLost,
		Message:   message,
	}
}

func (s *gameService) handleContinueGame(game *model.Game, eliminatedUser *userModel.User) *model.VoteResult {
	game.State = model.GameStatePlaying
	game.RoundNumber++
	game.VoteCount = make(map[string]int)
	game.VotedUsers = make(map[string]string)
	message := fmt.Sprintf("%s was not the impostor. Continue playing!", eliminatedUser.Nickname)

	s.hub.BroadcastToRoom(game.RoomID, ws.EventRoomUpdate, map[string]interface{}{
		"round_number": game.RoundNumber,
		"message":      message,
	})

	return &model.VoteResult{
		GameState: model.GameStatePlaying,
		Message:   message,
	}
}

func (s *gameService) EndGame(ctx context.Context, gameID string) error {
	game, err := s.gameRepo.GetByID(ctx, gameID)
	if err != nil {
		return err
	}

	game.State = model.GameStateLost
	return s.gameRepo.Update(ctx, game)
}

func (s *gameService) LeaveGame(ctx context.Context, gameID string, req *model.LeaveGameRequest) error {
	game, err := s.gameRepo.GetByID(ctx, gameID)
	if err != nil {
		return fmt.Errorf("game not found: %w", err)
	}

	if game.State != model.GameStatePlaying && game.State != model.GameStateVoting && game.State != model.GameStatePaused {
		return fmt.Errorf("game is not active")
	}

	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.RoomID != game.RoomID {
		return fmt.Errorf("user is not in this game")
	}

	reason := fmt.Sprintf("%s left the game", user.Nickname)
	return s.CancelGame(ctx, gameID, reason)
}

func (s *gameService) CancelGame(ctx context.Context, gameID string, reason string) error {
	game, err := s.gameRepo.GetByID(ctx, gameID)
	if err != nil {
		return fmt.Errorf("game not found: %w", err)
	}

	s.clearAllTimersForGame(gameID)

	game.State = model.GameStateCancelled

	s.hub.BroadcastToRoom(game.RoomID, ws.EventGameCancelled, map[string]interface{}{
		"game_id":     gameID,
		"reason":      reason,
		"impostor_id": game.ImpostorID,
		"word":        game.CurrentWord,
	})

	users, err := s.userRepo.GetByRoomID(ctx, game.RoomID)
	if err != nil {
		log.Printf("Failed to get users for cleanup: %v", err)
	} else {
		for _, user := range users {
			if delErr := s.userRepo.Delete(ctx, user.ID); delErr != nil {
				log.Printf("Failed to delete user %s: %v", user.ID, delErr)
			}
		}
	}

	if err := s.roomRepo.Delete(ctx, game.RoomID); err != nil {
		log.Printf("Failed to delete room %s: %v", game.RoomID, err)
	}

	if err := s.gameRepo.Delete(ctx, gameID); err != nil {
		log.Printf("Failed to delete game %s: %v", gameID, err)
	}

	log.Printf("Game %s cancelled: %s", gameID, reason)
	return nil
}

func (s *gameService) HandleDisconnect(clientID, roomID string) {
	ctx := context.Background()

	game, err := s.gameRepo.GetByRoomID(ctx, roomID)
	if err != nil {
		s.startLobbyDisconnectTimer(clientID, roomID)
		return
	}

	if game.State != model.GameStatePlaying && game.State != model.GameStateVoting && game.State != model.GameStatePaused {
		s.startLobbyDisconnectTimer(clientID, roomID)
		return
	}

	user, err := s.userRepo.GetByID(ctx, clientID)
	if err != nil {
		log.Printf("Failed to get user %s for disconnect handling: %v", clientID, err)
		return
	}

	if !user.IsAlive {
		return
	}

	s.startDisconnectTimer(game, user)
}

func (s *gameService) startDisconnectTimer(game *model.Game, user *userModel.User) {
	s.timersMu.Lock()
	defer s.timersMu.Unlock()

	if existing, exists := s.disconnectTimers[user.ID]; exists {
		existing.timer.Stop()
		delete(s.disconnectTimers, user.ID)
	}

	if game.DisconnectedUsers == nil {
		game.DisconnectedUsers = make(map[string]*model.DisconnectedUser)
	}

	disconnectedUser := &model.DisconnectedUser{
		UserID:       user.ID,
		Nickname:     user.Nickname,
		DisconnectAt: time.Now(),
	}
	game.DisconnectedUsers[user.ID] = disconnectedUser

	previousState := game.State
	if game.State != model.GameStatePaused {
		game.PreviousState = game.State
		game.State = model.GameStatePaused
	}

	ctx := context.Background()
	if err := s.gameRepo.Update(ctx, game); err != nil {
		log.Printf("Failed to update game state for disconnect: %v", err)
	}

	s.hub.BroadcastToRoom(game.RoomID, ws.EventUserDisconnected, map[string]interface{}{
		"user_id":         user.ID,
		"nickname":        user.Nickname,
		"timeout_seconds": int(ReconnectTimeout.Seconds()),
		"disconnect_at":   disconnectedUser.DisconnectAt,
		"previous_state":  previousState,
	})

	timer := time.AfterFunc(ReconnectTimeout, func() {
		s.onDisconnectTimeout(game.ID, user.ID, user.Nickname)
	})

	s.disconnectTimers[user.ID] = &disconnectTimer{
		timer:    timer,
		gameID:   game.ID,
		userID:   user.ID,
		nickname: user.Nickname,
	}

	log.Printf("Started %v disconnect timer for user %s in game %s", ReconnectTimeout, user.ID, game.ID)
}

func (s *gameService) onDisconnectTimeout(gameID, userID, nickname string) {
	s.timersMu.Lock()
	delete(s.disconnectTimers, userID)
	s.timersMu.Unlock()

	ctx := context.Background()
	reason := fmt.Sprintf("%s failed to reconnect in time", nickname)

	if err := s.CancelGame(ctx, gameID, reason); err != nil {
		log.Printf("Failed to cancel game on disconnect timeout: %v", err)
	}
}

func (s *gameService) HandleReconnect(clientID, roomID, gameID string) {
	s.timersMu.Lock()
	dt, exists := s.disconnectTimers[clientID]
	if exists {
		dt.timer.Stop()
		delete(s.disconnectTimers, clientID)
	}
	s.timersMu.Unlock()

	if !exists {
		log.Printf("No active disconnect timer for user %s", clientID)
		return
	}

	ctx := context.Background()
	game, err := s.gameRepo.GetByID(ctx, gameID)
	if err != nil {
		log.Printf("Failed to get game for reconnect: %v", err)
		return
	}

	user, err := s.userRepo.GetByID(ctx, clientID)
	if err != nil {
		log.Printf("Failed to get user for reconnect: %v", err)
		return
	}

	delete(game.DisconnectedUsers, clientID)

	if len(game.DisconnectedUsers) == 0 && game.State == model.GameStatePaused {
		game.State = game.PreviousState
		game.PreviousState = ""
	}

	if err := s.gameRepo.Update(ctx, game); err != nil {
		log.Printf("Failed to update game state for reconnect: %v", err)
	}

	s.hub.BroadcastToRoom(roomID, ws.EventUserReconnected, map[string]interface{}{
		"user_id":    clientID,
		"nickname":   user.Nickname,
		"game_state": game.State,
	})

	s.sendGameStateToUser(game, user)

	log.Printf("User %s reconnected to game %s", clientID, gameID)
}

func (s *gameService) sendGameStateToUser(game *model.Game, user *userModel.User) {
	payload := map[string]interface{}{
		"game_id":      game.ID,
		"state":        game.State,
		"category":     game.Category,
		"round_number": game.RoundNumber,
		"impostor_id":  game.ImpostorID,
		"vote_count":   game.VoteCount,
		"voted_users":  game.VotedUsers,
	}

	if user.ID != game.ImpostorID {
		payload["current_word"] = game.CurrentWord
	}

	s.hub.SendToClient(user.ID, game.RoomID, ws.EventGameStarted, payload)
}

func (s *gameService) startLobbyDisconnectTimer(clientID, roomID string) {
	s.timersMu.Lock()
	defer s.timersMu.Unlock()

	if existing, ok := s.lobbyDisconnectTimers[clientID]; ok {
		existing.Stop()
	}
	timer := time.AfterFunc(LobbyReconnectTimeout, func() {
		s.onLobbyDisconnectTimeout(clientID, roomID)
	})
	s.lobbyDisconnectTimers[clientID] = timer
}

func (s *gameService) cancelLobbyTimer(clientID string) {
	s.timersMu.Lock()
	defer s.timersMu.Unlock()

	if timer, ok := s.lobbyDisconnectTimers[clientID]; ok {
		timer.Stop()
		delete(s.lobbyDisconnectTimers, clientID)
	}
}

func (s *gameService) onLobbyDisconnectTimeout(clientID, roomID string) {
	s.timersMu.Lock()
	delete(s.lobbyDisconnectTimers, clientID)
	s.timersMu.Unlock()

	ctx := context.Background()

	user, err := s.userRepo.GetByID(ctx, clientID)
	if err != nil {
		return // already removed
	}
	if user.RoomID != roomID {
		return // moved or already gone
	}

	if err := s.userRepo.Delete(ctx, clientID); err != nil {
		log.Printf("Lobby timeout: failed to delete user %s: %v", clientID, err)
		return
	}

	s.hub.BroadcastToRoom(roomID, ws.EventUserKicked, map[string]interface{}{
		"user_id":  clientID,
		"nickname": user.Nickname,
		"reason":   "disconnected",
	})
}

func (s *gameService) clearAllTimersForGame(gameID string) {
	s.timersMu.Lock()
	defer s.timersMu.Unlock()

	for userID, dt := range s.disconnectTimers {
		if dt.gameID == gameID {
			dt.timer.Stop()
			delete(s.disconnectTimers, userID)
		}
	}
}

func (s *gameService) getAliveUsers(ctx context.Context, roomID string) ([]*userModel.User, error) {
	users, err := s.userRepo.GetByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	alive := make([]*userModel.User, 0, len(users))
	for _, user := range users {
		if user.IsAlive {
			alive = append(alive, user)
		}
	}

	return alive, nil
}

func (s *gameService) countAlivePlayers(roomID string) int {
	users, err := s.userRepo.GetByRoomID(context.Background(), roomID)
	if err != nil {
		return 0
	}

	count := 0
	for _, user := range users {
		if user.IsAlive {
			count++
		}
	}
	return count
}

func (s *gameService) getMostVotedUser(voteCount map[string]int) string {
	maxVotes := 0
	var eliminatedID string

	for userID, votes := range voteCount {
		if votes > maxVotes {
			maxVotes = votes
			eliminatedID = userID
		}
	}

	return eliminatedID
}

func (s *gameService) ReturnToRoom(ctx context.Context, gameID string, req *model.ReturnToRoomRequest) error {
	game, err := s.gameRepo.GetByID(ctx, gameID)
	if err != nil {
		return fmt.Errorf("game not found: %w", err)
	}

	if game.State != model.GameStateWon && game.State != model.GameStateLost {
		return fmt.Errorf("game has not finished yet")
	}

	users, err := s.userRepo.GetByRoomID(ctx, game.RoomID)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	for _, user := range users {
		user.Role = userModel.RolePlayer
		user.IsAlive = false
		user.IsReady = false
		if err := s.userRepo.Update(ctx, user); err != nil {
			log.Printf("Failed to reset user %s: %v", user.ID, err)
		}
	}

	if err := s.gameRepo.Delete(ctx, gameID); err != nil {
		return fmt.Errorf("failed to clean up game: %w", err)
	}

	s.hub.BroadcastToRoom(game.RoomID, ws.EventRoomUpdate, map[string]interface{}{
		"room_id": game.RoomID,
		"message": "Game finished, back in lobby",
	})

	return nil
}
