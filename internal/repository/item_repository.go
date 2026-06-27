package repository

import (
	"auction-backend/internal/models"

	"gorm.io/gorm"
)

type ItemRepository struct {
	db *gorm.DB
}

func NewItemRepository(db *gorm.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) Create(item *models.Item) error {
	return r.db.Create(item).Error
}