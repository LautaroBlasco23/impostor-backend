package model

type Word struct {
	ID       int    `json:"id"`
	Text     string `json:"text"`
	Category string `json:"category"`
}

type CreateWordRequest struct {
	Text     string `json:"text" validate:"required,min=1"`
	Category string `json:"category" validate:"required"`
}
