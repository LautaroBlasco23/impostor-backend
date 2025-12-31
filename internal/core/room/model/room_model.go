package model

import "time"

type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	LeaderID  string    `json:"leader_id"`
	MaxUsers  int       `json:"max_users"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
	Category  string    `json:"category,omitempty"`
}

type CreateRoomRequest struct {
	Name     string `json:"name" validate:"required,min=3,max=50"`
	MaxUsers int    `json:"max_users" validate:"required,min=2,max=10"`
	LeaderID string `json:"leader_id" validate:"required"`
}

type SetCategoryRequest struct {
	Category string `json:"category" validate:"required"`
}
