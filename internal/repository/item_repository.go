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

func (r *ItemRepository) GetBySellerID(userID uint) ([]models.Item, error) {
	var items []models.Item
	err := r.db.Where("seller_id = ?", userID).Order("id DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}

	return items, err
}

func (r *ItemRepository) List(q string, page, limit int, status string) ([]models.Item, int64, error) {
	var (
		items []models.Item
		total int64
	)

	query := r.db.Model(&models.Item{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("title LIKE ? OR description LIKE ?", like, like)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (r *ItemRepository) CountAll() (int64, error) {
	var total int64
	err := r.db.Model(&models.Item{}).Count(&total).Error
	return total, err
}

func (r *ItemRepository) CountByStatus(status string) (int64, error) {
	var total int64
	err := r.db.Model(&models.Item{}).Where("status = ?", status).Count(&total).Error
	return total, err
}

func (r *ItemRepository) SumEndedRevenue() (float64, error) {
	var total float64
	err := r.db.Model(&models.Item{}).
		Where("status IN ?", []string{"ended", "finished", "sold", "closed"}).
		Select("COALESCE(SUM(CASE WHEN current_price > 0 THEN current_price ELSE price END), 0)").
		Scan(&total).Error
	return total, err
}

func (r *ItemRepository) Update(item *models.Item) error {
	return r.db.Save(item).Error
}

func (r *ItemRepository) Delete(id uint) error {
	return r.db.Delete(&models.Item{}, id).Error
}
