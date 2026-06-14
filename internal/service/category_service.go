package service

import (
	"github.com/NicoMartinns/gastano-menos/internal/domain"
	"github.com/NicoMartinns/gastano-menos/internal/repository"
)

type CategoryService struct {
	repo *repository.CategoryRepository
}

func NewCategoryService(repo *repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) GetAll(userID string) ([]domain.Category, error) {
	return s.repo.FindAll(userID)
}

func (s *CategoryService) Create(userID, name, categoryType string, parentID *string) (*domain.Category, error) {
	c := &domain.Category{
		UserID:   userID,
		Name:     name,
		Type:     domain.CategoryType(categoryType),
		ParentID: parentID,
	}
	if err := s.repo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) Delete(id string, userID string) error {
	return s.repo.Delete(id, userID)
}