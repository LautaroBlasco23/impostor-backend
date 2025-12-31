package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/LautaroBlasco23/impostor/internal/core/user/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByRoomID(ctx context.Context, roomID string) ([]*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
}

type userRepository struct {
	client *redis.Client
}

func NewUserRepository(client *redis.Client) UserRepository {
	return &userRepository{client: client}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("user:%s", user.ID)
	if err := r.client.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return err
	}

	roomKey := fmt.Sprintf("room:%s:users", user.RoomID)
	return r.client.SAdd(ctx, roomKey, user.ID).Err()
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	key := fmt.Sprintf("user:%s", id)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	var user model.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetByRoomID(ctx context.Context, roomID string) ([]*model.User, error) {
	roomKey := fmt.Sprintf("room:%s:users", roomID)
	userIDs, err := r.client.SMembers(ctx, roomKey).Result()
	if err != nil {
		return nil, err
	}

	users := make([]*model.User, 0, len(userIDs))
	for _, id := range userIDs {
		user, err := r.GetByID(ctx, id)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("user:%s", user.ID)
	return r.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	roomKey := fmt.Sprintf("room:%s:users", user.RoomID)
	if err := r.client.SRem(ctx, roomKey, id).Err(); err != nil {
		return err
	}

	key := fmt.Sprintf("user:%s", id)
	return r.client.Del(ctx, key).Err()
}
