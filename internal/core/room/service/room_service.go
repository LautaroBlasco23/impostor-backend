package service

import (
	"context"
	"fmt"
	"time"

	"github.com/LautaroBlasco23/impostor/internal/core/room/model"
	"github.com/LautaroBlasco23/impostor/internal/core/room/repository"
	ws "github.com/LautaroBlasco23/impostor/internal/websocket"
	"github.com/google/uuid"
)

type RoomService interface {
	CreateRoom(ctx context.Context, req *model.CreateRoomRequest) (*model.Room, error)
	GetRoom(ctx context.Context, id string) (*model.Room, error)
	GetAllRooms(ctx context.Context) ([]*model.Room, error)
	SetCategory(ctx context.Context, roomID, leaderID, category string) error
	DeleteRoom(ctx context.Context, id string) error
}

type roomService struct {
	repo repository.RoomRepository
	hub  *ws.Hub
}

func NewRoomService(repo repository.RoomRepository, hub *ws.Hub) RoomService {
	return &roomService{
		repo: repo,
		hub:  hub,
	}
}

func (s *roomService) CreateRoom(ctx context.Context, req *model.CreateRoomRequest) (*model.Room, error) {
	room := &model.Room{
		ID:        uuid.New().String(),
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

func (s *roomService) DeleteRoom(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
