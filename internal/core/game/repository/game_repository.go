package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LautaroBlasco23/impostor/internal/core/game/model"
	"github.com/redis/go-redis/v9"
)

type GameRepository interface {
	Create(ctx context.Context, game *model.Game) error
	GetByID(ctx context.Context, id string) (*model.Game, error)
	GetByRoomID(ctx context.Context, roomID string) (*model.Game, error)
	Update(ctx context.Context, game *model.Game) error
	Delete(ctx context.Context, id string) error
}

type gameRepository struct {
	client *redis.Client
}

func NewGameRepository(client *redis.Client) GameRepository {
	return &gameRepository{client: client}
}

func (r *gameRepository) Create(ctx context.Context, game *model.Game) error {
	data, err := json.Marshal(game)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("game:%s", game.ID)
	if err := r.client.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return err
	}

	roomKey := fmt.Sprintf("room:%s:game", game.RoomID)
	return r.client.Set(ctx, roomKey, game.ID, 24*time.Hour).Err()
}

func (r *gameRepository) GetByID(ctx context.Context, id string) (*model.Game, error) {
	key := fmt.Sprintf("game:%s", id)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("game not found")
		}
		return nil, err
	}

	var game model.Game
	if err := json.Unmarshal(data, &game); err != nil {
		return nil, err
	}

	return &game, nil
}

func (r *gameRepository) GetByRoomID(ctx context.Context, roomID string) (*model.Game, error) {
	roomKey := fmt.Sprintf("room:%s:game", roomID)
	gameID, err := r.client.Get(ctx, roomKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("no active game in room")
		}
		return nil, err
	}

	return r.GetByID(ctx, gameID)
}

func (r *gameRepository) Update(ctx context.Context, game *model.Game) error {
	data, err := json.Marshal(game)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("game:%s", game.ID)
	return r.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (r *gameRepository) Delete(ctx context.Context, id string) error {
	game, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	roomKey := fmt.Sprintf("room:%s:game", game.RoomID)
	if err := r.client.Del(ctx, roomKey).Err(); err != nil {
		return err
	}

	key := fmt.Sprintf("game:%s", id)
	return r.client.Del(ctx, key).Err()
}
