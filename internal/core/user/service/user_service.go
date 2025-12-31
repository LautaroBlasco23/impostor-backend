package service

import (
	"context"
	"fmt"
	"time"

	"github.com/LautaroBlasco23/impostor/internal/core/user/model"
	"github.com/LautaroBlasco23/impostor/internal/core/user/repository"
	ws "github.com/LautaroBlasco23/impostor/internal/websocket"
	"github.com/google/uuid"
)

type UserService interface {
	CreateUser(ctx context.Context, req *model.CreateUserRequest) (*model.User, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
	GetUsersByRoom(ctx context.Context, roomID string) ([]*model.User, error)
	JoinRoom(ctx context.Context, userID, roomID string) error
	ToggleReady(ctx context.Context, userID string) error
	CheckAllReady(ctx context.Context, roomID string) (bool, error)
	DeleteUser(ctx context.Context, id string) error
}

type userService struct {
	repo repository.UserRepository
	hub  *ws.Hub
}

func NewUserService(repo repository.UserRepository, hub *ws.Hub) UserService {
	return &userService{
		repo: repo,
		hub:  hub,
	}
}

func (s *userService) CreateUser(ctx context.Context, req *model.CreateUserRequest) (*model.User, error) {
	user := &model.User{
		ID:        uuid.New().String(),
		Nickname:  req.Nickname,
		Role:      model.RolePlayer,
		IsReady:   false,
		IsAlive:   true,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) GetUser(ctx context.Context, id string) (*model.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *userService) GetUsersByRoom(ctx context.Context, roomID string) ([]*model.User, error) {
	return s.repo.GetByRoomID(ctx, roomID)
}

func (s *userService) JoinRoom(ctx context.Context, userID, roomID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.RoomID != "" {
		return fmt.Errorf("user already in a room")
	}

	user.RoomID = roomID
	if err := s.repo.Update(ctx, user); err != nil {
		return err
	}

	s.hub.UpdateClientRoom(userID, roomID)

	s.hub.BroadcastToRoom(roomID, ws.EventUserJoined, map[string]interface{}{
		"user_id":  userID,
		"nickname": user.Nickname,
		"room_id":  roomID,
	})

	return nil
}

func (s *userService) ToggleReady(ctx context.Context, userID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	user.IsReady = !user.IsReady
	if err := s.repo.Update(ctx, user); err != nil {
		return err
	}

	s.hub.BroadcastToRoom(user.RoomID, ws.EventUserReady, map[string]interface{}{
		"user_id":  userID,
		"nickname": user.Nickname,
		"is_ready": user.IsReady,
	})

	return nil
}

func (s *userService) CheckAllReady(ctx context.Context, roomID string) (bool, error) {
	users, err := s.repo.GetByRoomID(ctx, roomID)
	if err != nil {
		return false, err
	}

	if len(users) < 2 {
		return false, nil
	}

	for _, user := range users {
		if !user.IsReady {
			return false, nil
		}
	}

	return true, nil
}

func (s *userService) DeleteUser(ctx context.Context, id string) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if user.RoomID != "" {
		s.hub.BroadcastToRoom(user.RoomID, ws.EventUserLeft, map[string]interface{}{
			"user_id":  id,
			"nickname": user.Nickname,
		})
	}

	return s.repo.Delete(ctx, id)
}
