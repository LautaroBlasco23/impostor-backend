package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/LautaroBlasco23/impostor/internal/core/game/model"
	"github.com/LautaroBlasco23/impostor/internal/core/game/repository"
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
	room, err := s.roomRepo.GetByID(ctx, req.RoomID)
	if err != nil {
		return nil, fmt.Errorf("room not found: %w", err)
	}

	if !room.IsActive {
		return nil, fmt.Errorf("room is not active")
	}

	if room.Category == "" {
		return nil, fmt.Errorf("room category not set")
	}

	users, err := s.userRepo.GetByRoomID(ctx, req.RoomID)
	if err != nil {
		return nil, err
	}

	if len(users) < 2 {
		return nil, fmt.Errorf("need at least 2 users to start game")
	}

	for _, user := range users {
		if !user.IsReady {
			return nil, fmt.Errorf("all users must be ready")
		}
	}

	words, err := s.wordRepo.GetRandomByCategory(ctx, room.Category, 1)
	if err != nil || len(words) == 0 {
		return nil, fmt.Errorf("no words found for category: %s", room.Category)
	}

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
			return nil, err
		}
	}

	game := &model.Game{
		ID:          uuid.New().String(),
		RoomID:      req.RoomID,
		State:       model.GameStatePlaying,
		ImpostorID:  impostorID,
		CurrentWord: words[0].Text,
		Category:    room.Category,
		VoteCount:   make(map[string]int),
		VotedUsers:  make(map[string]string),
		RoundNumber: 1,
		CreatedAt:   time.Now(),
	}

	if err := s.gameRepo.Create(ctx, game); err != nil {
		return nil, err
	}

	// Send personalized events to each client
	for _, user := range users {
		payload := map[string]interface{}{
			"game_id":      game.ID,
			"category":     game.Category,
			"round_number": game.RoundNumber,
			"impostor_id":  game.ImpostorID,
		}

		// Only send word to non-impostors
		if user.ID != game.ImpostorID {
			payload["current_word"] = game.CurrentWord
		}

		s.hub.SendToClient(user.ID, game.RoomID, ws.EventGameStarted, payload)
	}

	return game, nil
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

	if game.State != model.GameStatePlaying && game.State != model.GameStateVoting {
		return nil, fmt.Errorf("game is not in playing state")
	}

	voter, err := s.userRepo.GetByID(ctx, req.VoterID)
	if err != nil {
		return nil, fmt.Errorf("voter not found: %w", err)
	}

	if !voter.IsAlive {
		return nil, fmt.Errorf("dead users cannot vote")
	}

	if voter.RoomID != game.RoomID {
		return nil, fmt.Errorf("voter is not in the game room")
	}

	target, err := s.userRepo.GetByID(ctx, req.TargetID)
	if err != nil {
		return nil, fmt.Errorf("target not found: %w", err)
	}

	if !target.IsAlive {
		return nil, fmt.Errorf("cannot vote for dead user")
	}

	if _, hasVoted := game.VotedUsers[req.VoterID]; hasVoted {
		return nil, fmt.Errorf("user has already voted this round")
	}

	game.VotedUsers[req.VoterID] = req.TargetID
	game.VoteCount[req.TargetID]++
	game.State = model.GameStateVoting

	s.hub.BroadcastToRoom(game.RoomID, ws.EventUserVoted, map[string]interface{}{
		"voter_id":      req.VoterID,
		"voter_name":    voter.Nickname,
		"target_id":     req.TargetID,
		"target_name":   target.Nickname,
		"votes_cast":    len(game.VotedUsers),
		"total_players": s.countAlivePlayers(game.RoomID),
	})

	aliveUsers, err := s.getAliveUsers(ctx, game.RoomID)
	if err != nil {
		return nil, err
	}

	if len(game.VotedUsers) < len(aliveUsers) {
		if err := s.gameRepo.Update(ctx, game); err != nil {
			return nil, err
		}
		return &model.VoteResult{
			GameState: model.GameStateVoting,
			Message:   fmt.Sprintf("Waiting for %d more votes", len(aliveUsers)-len(game.VotedUsers)),
		}, nil
	}

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

	result := &model.VoteResult{
		EliminatedUserID: eliminatedID,
		WasImpostor:      wasImpostor,
	}

	s.hub.BroadcastToRoom(game.RoomID, ws.EventUserEliminated, map[string]interface{}{
		"user_id":      eliminatedID,
		"nickname":     eliminatedUser.Nickname,
		"was_impostor": wasImpostor,
		"vote_count":   game.VoteCount[eliminatedID],
	})

	if wasImpostor {
		game.State = model.GameStateWon
		result.GameState = model.GameStateWon
		result.Message = fmt.Sprintf("Players win! %s was the impostor!", eliminatedUser.Nickname)

		s.hub.BroadcastToRoom(game.RoomID, ws.EventGameWon, map[string]interface{}{
			"impostor_id":   eliminatedID,
			"impostor_name": eliminatedUser.Nickname,
			"message":       result.Message,
		})
	} else {
		aliveAfterVote, err := s.getAliveUsers(ctx, game.RoomID)
		if err != nil {
			return nil, err
		}

		if len(aliveAfterVote) <= 1 {
			game.State = model.GameStateLost
			result.GameState = model.GameStateLost
			result.Message = "Impostor wins! Not enough players remaining!"

			impostor, _ := s.userRepo.GetByID(ctx, game.ImpostorID)
			impostorName := "Unknown"
			if impostor != nil {
				impostorName = impostor.Nickname
			}

			s.hub.BroadcastToRoom(game.RoomID, ws.EventGameLost, map[string]interface{}{
				"impostor_id":   game.ImpostorID,
				"impostor_name": impostorName,
				"message":       result.Message,
			})
		} else {
			game.State = model.GameStatePlaying
			game.RoundNumber++
			game.VoteCount = make(map[string]int)
			game.VotedUsers = make(map[string]string)
			result.GameState = model.GameStatePlaying
			result.Message = fmt.Sprintf("%s was not the impostor. Continue playing!", eliminatedUser.Nickname)

			s.hub.BroadcastToRoom(game.RoomID, ws.EventRoomUpdate, map[string]interface{}{
				"round_number": game.RoundNumber,
				"message":      result.Message,
			})
		}
	}

	if err := s.gameRepo.Update(ctx, game); err != nil {
		return nil, err
	}

	return result, nil
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

	alive := make([]*userModel.User, 0)
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
