package model

import "time"

type UserRole string

const (
	RolePlayer   UserRole = "player"
	RoleImpostor UserRole = "impostor"
)

type User struct {
	ID        string    `json:"id"`
	Nickname  string    `json:"nickname"`
	RoomID    string    `json:"room_id"`
	Role      UserRole  `json:"role"`
	IsReady   bool      `json:"is_ready"`
	IsAlive   bool      `json:"is_alive"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
	Nickname string `json:"nickname" validate:"required,min=3,max=20"`
}

type JoinRoomRequest struct {
	RoomID string `json:"room_id" validate:"required"`
}
