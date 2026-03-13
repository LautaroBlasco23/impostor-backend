package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/LautaroBlasco23/impostor/internal/core/room/model"
	"github.com/LautaroBlasco23/impostor/internal/core/room/repository"
	userRepo "github.com/LautaroBlasco23/impostor/internal/core/user/repository"
	ws "github.com/LautaroBlasco23/impostor/internal/websocket"
)

type RoomService interface {
	CreateRoom(ctx context.Context, req *model.CreateRoomRequest) (*model.Room, error)
	GetRoom(ctx context.Context, id string) (*model.Room, error)
	GetAllRooms(ctx context.Context) ([]*model.Room, error)
	SetCategory(ctx context.Context, roomID, leaderID, category string) error
	DeleteRoom(ctx context.Context, id, leaderID string) error
	KickUser(ctx context.Context, roomID, leaderID, targetUserID string) error
}

type roomService struct {
	repo     repository.RoomRepository
	userRepo userRepo.UserRepository
	hub      *ws.Hub
}

func NewRoomService(repo repository.RoomRepository, userRepo userRepo.UserRepository, hub *ws.Hub) RoomService {
	return &roomService{
		repo: repo,
		userRepo: userRepo,
		hub: hub,
	}
}

func (s *roomService) CreateRoom(ctx context.Context, req *model.CreateRoomRequest) (*model.Room, error) {
	id, err := s.repo.NextID(ctx)
	if err != nil {
		return nil, err
	}
	room := &model.Room{
		ID:        id,
		Name:      req.Name,
		LeaderID:  req.LeaderID,
		MaxUsers:  req.MaxUsers,
		CreatedAt: time.Now(),
		IsActive:  true,
	}
	if err := s.repo.Create(ctx, room); err != nil {
		return nil, err
	}
	return room, nil
}

func (s *roomService) GetRoom(ctx context.Context, id string) (*model.Room, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *roomService) GetAllRooms(ctx context.Context) ([]*model.Room, error) {
	return s.repo.GetAll(ctx)
}

func (s *roomService) SetCategory(ctx context.Context, roomID, leaderID, category string) error {
	room, err := s.repo.GetByID(ctx, roomID)
	if err != nil {
		return err
	}

	if room.LeaderID != leaderID {
		return fmt.Errorf("only room leader can set category")
	}

	room.Category = category
	if err := s.repo.Update(ctx, room); err != nil {
		return err
	}

	s.hub.BroadcastToRoom(roomID, ws.EventCategorySet, map[string]interface{}{
		"category": category,
		"room_id":  roomID,
	})

	return nil
}

func (s *roomService) DeleteRoom(ctx context.Context, id, leaderID string) error {
	room, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("room not found: %w", err)
	}

	if room.LeaderID != leaderID {
		return fmt.Errorf("only the room owner can delete this room")
	}

	users, err := s.userRepo.GetByRoomID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get room users: %w", err)
	}

	for _, user := range users {
		if err := s.userRepo.Delete(ctx, user.ID); err != nil {
			log.Printf("Failed to delete user %s: %v", user.ID, err)
		}
	}

	s.hub.BroadcastToRoom(id, ws.EventGameCancelled, map[string]interface{}{
		"room_id": id,
		"reason":  "Room closed by owner",
	})

	return s.repo.Delete(ctx, id)
}

func (s *roomService) KickUser(ctx context.Context, roomID, leaderID, targetUserID string) error {
	room, err := s.repo.GetByID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("room not found: %w", err)
	}
	if room.LeaderID != leaderID {
		return fmt.Errorf("only the room leader can kick players")
	}
	if leaderID == targetUserID {
		return fmt.Errorf("cannot kick yourself")
	}

	user, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if user.RoomID != roomID {
		return fmt.Errorf("user is not in this room")
	}

	if err := s.userRepo.Delete(ctx, targetUserID); err != nil {
		return fmt.Errorf("failed to remove user: %w", err)
	}

	s.hub.BroadcastToRoom(roomID, ws.EventUserKicked, map[string]interface{}{
		"user_id":  targetUserID,
		"nickname": user.Nickname,
		"reason":   "kicked",
	})
	return nil
}

