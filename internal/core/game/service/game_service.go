package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/LautaroBlasco23/impostor/internal/core/game/model"
	"github.com/LautaroBlasco23/impostor/internal/core/game/repository"
	roomModel "github.com/LautaroBlasco23/impostor/internal/core/room/model"
	roomRepo "github.com/LautaroBlasco23/impostor/internal/core/room/repository"
	userModel "github.com/LautaroBlasco23/impostor/internal/core/user/model"
	userRepo "github.com/LautaroBlasco23/impostor/internal/core/user/repository"
	wordRepo "github.com/LautaroBlasco23/impostor/internal/core/word/repository"
	ws "github.com/LautaroBlasco23/impostor/internal/websocket"
	"github.com/google/uuid"
)

type GameService interface {
	StartGame(ctx context.Context, req *model.StartGameRequest) (*model.Game, error)
	GetGame(ctx context.Context, id string) (*model.Game, error)
	GetGameByRoom(ctx context.Context, roomID string) (*model.Game, error)
	Vote(ctx context.Context, req *model.VoteRequest) (*model.VoteResult, error)
	EndGame(ctx context.Context, gameID string) error
}

type gameService struct {
	gameRepo repository.GameRepository
	roomRepo roomRepo.RoomRepository
	userRepo userRepo.UserRepository
	wordRepo wordRepo.WordRepository
	hub      *ws.Hub
}

func NewGameService(
	gameRepo repository.GameRepository,
	roomRepo roomRepo.RoomRepository,
	userRepo userRepo.UserRepository,
	wordRepo wordRepo.WordRepository,
	hub *ws.Hub,
) GameService {
	return &gameService{
		gameRepo: gameRepo,
		roomRepo: roomRepo,
		userRepo: userRepo,
		wordRepo: wordRepo,
		hub:      hub,
	}
}

func (s *gameService) StartGame(ctx context.Context, req *model.StartGameRequest) (*model.Game, error) {
	room, users, err := s.validateAndGetStartGameData(ctx, req.RoomID)
	if err != nil {
		return nil, err
	}

	words, err := s.wordRepo.GetRandomByCategory(ctx, room.Category, 1)
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

	if len(users) < 2 {
		return nil, nil, fmt.Errorf("need at least 2 users to start game")
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
		ID:          uuid.New().String(),
		RoomID:      roomID,
		State:       model.GameStatePlaying,
		ImpostorID:  impostorID,
		CurrentWord: word,
		Category:    category,
		VoteCount:   make(map[string]int),
		VotedUsers:  make(map[string]string),
		RoundNumber: 1,
		CreatedAt:   time.Now(),
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

	if len(aliveAfterVote) <= 1 {
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
