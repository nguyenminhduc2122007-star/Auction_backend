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

func (u UserType) IsValid() bool {
	switch u {
	case UserTypeBidder, UserTypeSeller, UserTypeAdmin:
		return true
	default:
		return false
	}
}

func (u UserType) IsAdmin() bool {
	return u == UserTypeAdmin
}

func (u UserType) IsSeller() bool {
	return u == UserTypeSeller
}

func (u UserType) IsBidder() bool {
	return u == UserTypeBidder
}

type User struct {
	ID              uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Email           string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	FullName        string         `gorm:"type:varchar(255);not null" json:"full_name"`
	PasswordHash    string         `gorm:"type:varchar(255);not null" json:"-"`
	UserType        UserType       `gorm:"type:varchar(20);not null;default:'Bidder'" json:"user_type"`
	WalletBalance   float64        `gorm:"type:numeric(15,2);not null;default:0" json:"wallet_balance"`
	Phone           string         `gorm:"type:varchar(32)" json:"phone,omitempty"`
	ShippingAddress string         `gorm:"type:text" json:"shipping_address,omitempty"`
}

type Item struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Title        string         `gorm:"type:varchar(255);not null" json:"title"`
	Category     string         `gorm:"type:varchar(100);not null;default:General" json:"category"`
	Status       string         `gorm:"type:varchar(50);not null;default:pending;index" json:"status"`
	Price        float64        `gorm:"type:decimal(12,2);not null;default:0" json:"price"`
	Description  string         `gorm:"type:text" json:"description"`
	StartPrice   float64        `gorm:"type:decimal(12,2);not null;default:0" json:"start_price"`
	CurrentPrice float64        `gorm:"type:decimal(12,2);not null;default:0" json:"current_price"`
	BidStep      float64        `gorm:"type:decimal(12,2);not null;default:0" json:"bid_step"`
	SellerID     uint           `gorm:"not null;index" json:"seller_id"`
	Seller       User           `gorm:"foreignKey:SellerID" json:"-"`
	UserID       uint           `gorm:"not null;default:0" json:"user_id"`
}

// DTO cho F-005 (Trả về thông tin chi tiết Tab Đang đặt giá)
type UserBidItemDTO struct {
	AuctionID    uint    `json:"auction_id"`
	Title        string  `json:"title"`
	CurrentPrice float64 `json:"current_price"`
	UserMaxBid   float64 `json:"user_max_bid"`
	IsWinning    bool    `json:"is_winning"`
	Status       string  `json:"status"`
}