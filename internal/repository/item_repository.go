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

func (r *ItemRepository) GetByID(id uint) (*models.Item, error) {
	var item models.Item
	err := r.db.First(&item, id).Error
	return &item, err
}

func (r *ItemRepository) GetAll() ([]models.Item, error) {
	var items []models.Item
	err := r.db.Find(&items).Error
	return items, err
}

func (r *ItemRepository) Update(item *models.Item) error {
	return r.db.Save(item).Error
}

func (r *ItemRepository) Delete(id uint) error {
	return r.db.Delete(&models.Item{}, id).Error
}