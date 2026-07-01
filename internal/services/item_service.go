package services

import (
	"errors"

	"auction-backend/internal/models"
	"auction-backend/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrItemNotFound     = errors.New("item not found")
	ErrPermissionDenied = errors.New("permission denied: not the owner of this item")
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

func (s *ItemService) GetItemByID(id uint) (*models.Item, error) {
	item, err := s.itemRepo.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ItemService) GetAllItems() ([]models.Item, error) {
	return s.itemRepo.GetAll()
}

func (s *ItemService) UpdateItem(userID, itemID uint, title, description string, price float64) (*models.Item, error) {
	item, err := s.itemRepo.GetByID(itemID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}
	if item.UserID != userID {
		return nil, ErrPermissionDenied
	}
	if title == "" {
		return nil, errors.New("title is required")
	}
	if price <= 0 {
		return nil, errors.New("price must be greater than 0")
	}

	item.Title = title
	item.Description = description
	item.Price = price

	if err := s.itemRepo.Update(item); err != nil {
		return nil, errors.New("failed to update item")
	}
	return item, nil
}

func (s *ItemService) DeleteItem(userID, itemID uint) error {
	item, err := s.itemRepo.GetByID(itemID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrItemNotFound
	}
	if err != nil {
		return err
	}
	if item.UserID != userID {
		return ErrPermissionDenied
	}

	if err := s.itemRepo.Delete(itemID); err != nil {
		return errors.New("failed to delete item")
	}
	return nil
}