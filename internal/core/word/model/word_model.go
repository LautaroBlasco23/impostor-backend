package model

type Word struct {
	ID       int    `json:"id"`
	Text     string `json:"text"`
	Category string `json:"category"`
	Language string `json:"language"`
}
