package repository

import (
	"context"
	"log"

	"github.com/LautaroBlasco23/impostor/internal/core/word/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WordRepository interface {
	Create(ctx context.Context, word *model.Word) error
	GetByID(ctx context.Context, id int) (*model.Word, error)
	GetByCategory(ctx context.Context, category, language string) ([]*model.Word, error)
	GetRandomByCategory(ctx context.Context, category, language string, limit int) ([]*model.Word, error)
	GetAll(ctx context.Context, language string) ([]*model.Word, error)
	GetCategories(ctx context.Context, language string) ([]string, error)
	Delete(ctx context.Context, id int) error
}

type wordRepository struct {
	pool *pgxpool.Pool
}

func NewWordRepository(pool *pgxpool.Pool) WordRepository {
	return &wordRepository{pool: pool}
}

func (r *wordRepository) Create(ctx context.Context, word *model.Word) error {
	query := `INSERT INTO words (text, category, language) VALUES ($1, $2, $3) RETURNING id`
	return r.pool.QueryRow(ctx, query, word.Text, word.Category, word.Language).Scan(&word.ID)
}

func (r *wordRepository) GetByID(ctx context.Context, id int) (*model.Word, error) {
	query := `SELECT id, text, category, language FROM words WHERE id = $1`
	var word model.Word
	err := r.pool.QueryRow(ctx, query, id).Scan(&word.ID, &word.Text, &word.Category, &word.Language)
	if err != nil {
		return nil, err
	}
	return &word, nil
}

func (r *wordRepository) GetByCategory(ctx context.Context, category, language string) ([]*model.Word, error) {
	query := `SELECT id, text, category, language FROM words WHERE category = $1 AND language = $2`
	return r.scanWords(ctx, query, category, language)
}

func (r *wordRepository) GetRandomByCategory(ctx context.Context, category, language string, limit int) ([]*model.Word, error) {
	query := `SELECT id, text, category, language FROM words WHERE category = $1 AND language = $2 ORDER BY RANDOM() LIMIT $3`
	return r.scanWords(ctx, query, category, language, limit)
}

func (r *wordRepository) GetAll(ctx context.Context, language string) ([]*model.Word, error) {
	query := `SELECT id, text, category, language FROM words WHERE language = $1`
	return r.scanWords(ctx, query, language)
}

func (r *wordRepository) GetCategories(ctx context.Context, language string) ([]string, error) {
	log.Printf("[GetCategories] language param: %q", language)
	query := `SELECT DISTINCT category FROM words WHERE language = $1 ORDER BY category`
	rows, err := r.pool.Query(ctx, query, language)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	log.Printf("[GetCategories] found %d categories", len(categories))
	return categories, rows.Err()
}

func (r *wordRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM words WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *wordRepository) scanWords(ctx context.Context, query string, args ...any) ([]*model.Word, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []*model.Word
	for rows.Next() {
		var word model.Word
		if err := rows.Scan(&word.ID, &word.Text, &word.Category, &word.Language); err != nil {
			return nil, err
		}
		words = append(words, &word)
	}
	return words, rows.Err()
}
