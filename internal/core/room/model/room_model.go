package model

import "time"

type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	LeaderID  string    `json:"leader_id"`
	MaxUsers  int       `json:"max_users"`
	Category  string    `json:"category,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
}

type CreateRoomRequest struct {
	Name     string `json:"name" validate:"required"`
	LeaderID string `json:"leader_id" validate:"required"`
	MaxUsers int    `json:"max_users" validate:"required,min=2"`
}

type SetCategoryRequest struct {
	Category string `json:"category" validate:"required"`
	LeaderID string `json:"leader_id" validate:"required"`
}
