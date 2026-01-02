package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LautaroBlasco23/impostor/internal/core/room/model"
	"github.com/redis/go-redis/v9"
)

type RoomRepository interface {
	Create(ctx context.Context, room *model.Room) error
	GetByID(ctx context.Context, id string) (*model.Room, error)
	GetAll(ctx context.Context) ([]*model.Room, error)
	Update(ctx context.Context, room *model.Room) error
	Delete(ctx context.Context, id string) error
}

type roomRepository struct {
	client *redis.Client
}

func NewRoomRepository(client *redis.Client) RoomRepository {
	return &roomRepository{client: client}
}

func (r *roomRepository) Create(ctx context.Context, room *model.Room) error {
	data, err := json.Marshal(room)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("room:%s", room.ID)
	return r.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (r *roomRepository) GetByID(ctx context.Context, id string) (*model.Room, error) {
	key := fmt.Sprintf("room:%s", id)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("room not found")
		}
		return nil, err
	}

	var room model.Room
	if err := json.Unmarshal(data, &room); err != nil {
		return nil, err
	}

	return &room, nil
}

func (r *roomRepository) GetAll(ctx context.Context) ([]*model.Room, error) {
	keys, err := r.client.Keys(ctx, "room:*").Result()
	if err != nil {
		return nil, err
	}

	rooms := make([]*model.Room, 0, len(keys))
	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var room model.Room
		if err := json.Unmarshal(data, &room); err != nil {
			continue
		}
		rooms = append(rooms, &room)
	}

	return rooms, nil
}

func (r *roomRepository) Update(ctx context.Context, room *model.Room) error {
	data, err := json.Marshal(room)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("room:%s", room.ID)
	return r.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (r *roomRepository) Delete(ctx context.Context, id string) error {
	key := fmt.Sprintf("room:%s", id)
	return r.client.Del(ctx, key).Err()
}
