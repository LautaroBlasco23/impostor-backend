package service

import (
	"context"

	"github.com/LautaroBlasco23/impostor/internal/core/word/model"
	"github.com/LautaroBlasco23/impostor/internal/core/word/repository"
)

type WordService interface {
	CreateWord(ctx context.Context, req *model.CreateWordRequest) (*model.Word, error)
	GetWord(ctx context.Context, id int) (*model.Word, error)
	GetWordsByCategory(ctx context.Context, category string) ([]*model.Word, error)
	GetRandomWords(ctx context.Context, category string, limit int) ([]*model.Word, error)
	GetAllWords(ctx context.Context) ([]*model.Word, error)
	GetCategories(ctx context.Context) ([]string, error)
	DeleteWord(ctx context.Context, id int) error
}

type wordService struct {
	repo repository.WordRepository
}

func NewWordService(repo repository.WordRepository) WordService {
	return &wordService{repo: repo}
}

func (s *wordService) CreateWord(ctx context.Context, req *model.CreateWordRequest) (*model.Word, error) {
	word := &model.Word{
		Text:     req.Text,
		Category: req.Category,
	}
	if err := s.repo.Create(ctx, word); err != nil {
		return nil, err
	}
	return word, nil
}

func (s *wordService) GetWord(ctx context.Context, id int) (*model.Word, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *wordService) GetWordsByCategory(ctx context.Context, category string) ([]*model.Word, error) {
	return s.repo.GetByCategory(ctx, category)
}

func (s *wordService) GetRandomWords(ctx context.Context, category string, limit int) ([]*model.Word, error) {
	return s.repo.GetRandomByCategory(ctx, category, limit)
}

func (s *wordService) GetAllWords(ctx context.Context) ([]*model.Word, error) {
	return s.repo.GetAll(ctx)
}

func (s *wordService) GetCategories(ctx context.Context) ([]string, error) {
	return s.repo.GetCategories(ctx)
}

func (s *wordService) DeleteWord(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
