package services

import (
	"errors"

	"auction-backend/internal/models"
	"auction-backend/internal/repository"
)

type ItemService struct {
	itemRepo *repository.ItemRepository
}

func NewItemService(itemRepo *repository.ItemRepository) *ItemService {
	return &ItemService{itemRepo: itemRepo}
}

func (s *ItemService) CreateItem(userID uint, title, description string, price float64) (*models.Item, error) {
	if title == "" {
		return nil, errors.New("title is required")
	}
	if price <= 0 {
		return nil, errors.New("price must be greater than 0")
	}

	item := &models.Item{
		Title:       title,
		Description: description,
		Price:       price,
		UserID:      userID,
	}

	if err := s.itemRepo.Create(item); err != nil {
		return nil, errors.New("failed to create item")
	}

	return item, nil
}