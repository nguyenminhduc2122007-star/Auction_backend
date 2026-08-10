package repository

import (
	"auction-backend/internal/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) CountAll() (int64, error) {
	var total int64
	err := r.db.Model(&models.User{}).Count(&total).Error
	return total, err
}

// CountByRole đếm số lượng người dùng theo UserType (dùng để kiểm tra số lượng Admin)
func (r *UserRepository) CountByRole(role models.UserType) (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("user_type = ?", role).Count(&count).Error
	return count, err
}

func (r *UserRepository) GetAll() ([]models.User, error) {
	var users []models.User
	err := r.db.Order("id DESC").Find(&users).Error
	return users, err
}

func (r *UserRepository) UpdateRole(id uint, role models.UserType) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	
	user.UserType = role
	if err := r.db.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}