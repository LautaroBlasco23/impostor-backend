package model

import "time"

type GameState string

const (
	GameStateWaiting GameState = "waiting"
	GameStatePlaying GameState = "playing"
	GameStateVoting  GameState = "voting"
	GameStateWon     GameState = "won"
	GameStateLost    GameState = "lost"
)

type Game struct {
	ID          string            `json:"id"`
	RoomID      string            `json:"room_id"`
	State       GameState         `json:"state"`
	ImpostorID  string            `json:"impostor_id,omitempty"`
	CurrentWord string            `json:"current_word,omitempty"`
	Category    string            `json:"category"`
	VoteCount   map[string]int    `json:"vote_count,omitempty"`
	VotedUsers  map[string]string `json:"voted_users,omitempty"`
	RoundNumber int               `json:"round_number"`
	CreatedAt   time.Time         `json:"created_at"`
}

type StartGameRequest struct {
	RoomID string `json:"room_id" validate:"required"`
}

type VoteRequest struct {
	GameID   string `json:"game_id" validate:"required"`
	VoterID  string `json:"voter_id" validate:"required"`
	TargetID string `json:"target_id" validate:"required"`
}

type VoteResult struct {
	EliminatedUserID string    `json:"eliminated_user_id"`
	WasImpostor      bool      `json:"was_impostor"`
	GameState        GameState `json:"game_state"`
	Message          string    `json:"message"`
}
