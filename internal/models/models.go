package models

import (
	"time"

	"gorm.io/gorm"
)

type UserType string

const (
	UserTypeBidder UserType = "Bidder"
	UserTypeSeller UserType = "Seller"
	UserTypeAdmin  UserType = "Admin"
)

type User struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Email        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	FullName     string         `gorm:"type:varchar(255);not null" json:"full_name"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	UserType     UserType       `gorm:"type:varchar(20);not null" json:"user_type"`
}

type Item struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"` // Kích hoạt Soft Delete giống User
	Title       string         `gorm:"type:varchar(255);not null" json:"title"`
	Description string         `gorm:"type:text" json:"description"`
	Price       float64        `gorm:"type:numeric(10,2);not null;default:0.00" json:"price"`
	
	// Khóa ngoại liên kết tới bảng Users
	UserID      uint           `gorm:"not null" json:"user_id"`
	
	// Cấu hình Quan hệ (Relationship) và tự động tạo Ràng buộc (Constraint) ở DB
	User        User           `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
}

