package repository

import (
	"context"
	"github.com/LautaroBlasco23/impostor/internal/core/word/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WordRepository interface {
	Create(ctx context.Context, word *model.Word) error
	GetByID(ctx context.Context, id int) (*model.Word, error)
	GetByCategory(ctx context.Context, category string) ([]*model.Word, error)
	GetRandomByCategory(ctx context.Context, category string, limit int) ([]*model.Word, error)
	GetAll(ctx context.Context) ([]*model.Word, error)
	GetCategories(ctx context.Context) ([]string, error)
	Delete(ctx context.Context, id int) error
}

type wordRepository struct {
	pool *pgxpool.Pool
}

func NewWordRepository(pool *pgxpool.Pool) WordRepository {
	return &wordRepository{pool: pool}
}

func (r *wordRepository) Create(ctx context.Context, word *model.Word) error {
	query := `INSERT INTO words (text, category) VALUES ($1, $2) RETURNING id`
	return r.pool.QueryRow(ctx, query, word.Text, word.Category).Scan(&word.ID)
}

func (r *wordRepository) GetByID(ctx context.Context, id int) (*model.Word, error) {
	query := `SELECT id, text, category FROM words WHERE id = $1`
	var word model.Word
	err := r.pool.QueryRow(ctx, query, id).Scan(&word.ID, &word.Text, &word.Category)
	if err != nil {
		return nil, err
	}
	return &word, nil
}

func (r *wordRepository) GetByCategory(ctx context.Context, category string) ([]*model.Word, error) {
	query := `SELECT id, text, category FROM words WHERE category = $1`
	rows, err := r.pool.Query(ctx, query, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	words := make([]*model.Word, 0)
	for rows.Next() {
		var word model.Word
		if err := rows.Scan(&word.ID, &word.Text, &word.Category); err != nil {
			continue
		}
		words = append(words, &word)
	}
	return words, nil
}

func (r *wordRepository) GetRandomByCategory(ctx context.Context, category string, limit int) ([]*model.Word, error) {
	query := `SELECT id, text, category FROM words WHERE category = $1 ORDER BY RANDOM() LIMIT $2`
	rows, err := r.pool.Query(ctx, query, category, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	words := make([]*model.Word, 0)
	for rows.Next() {
		var word model.Word
		if err := rows.Scan(&word.ID, &word.Text, &word.Category); err != nil {
			continue
		}
		words = append(words, &word)
	}
	return words, nil
}

func (r *wordRepository) GetAll(ctx context.Context) ([]*model.Word, error) {
	query := `SELECT id, text, category FROM words`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	words := make([]*model.Word, 0)
	for rows.Next() {
		var word model.Word
		if err := rows.Scan(&word.ID, &word.Text, &word.Category); err != nil {
			continue
		}
		words = append(words, &word)
	}
	return words, nil
}

func (r *wordRepository) GetCategories(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT category FROM words ORDER BY category`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]string, 0)
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			continue
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (r *wordRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM words WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
