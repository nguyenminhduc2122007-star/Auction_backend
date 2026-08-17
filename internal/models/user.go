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
	IsSuspicious    bool           `gorm:"default:false" json:"is_suspicious"` // Cờ đánh dấu Spam/Giá ảo
}

// Bảng mới: Lịch sử bị cảnh cáo của User
type UserWarning struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
	Reason    string    `gorm:"type:text;not null" json:"reason"`
	CreatedBy uint      `gorm:"not null" json:"created_by"` // Admin ID thực hiện cảnh báo
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

// DTO trả về danh sách Cảnh cáo trong Drawer
type UserWarningDTO struct {
	ID        uint      `json:"id"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// DTO tổng hợp chi tiết User cho Drawer (Target 4)
type UserSummaryDTO struct {
	ID                  uint             `json:"id"`
	FullName            string           `json:"full_name"`
	Email               string           `json:"email"`
	UserType            UserType         `json:"user_type"`
	IsSuspicious        bool             `json:"is_suspicious"`
	TotalSpent          float64          `json:"total_spent"`
	WonAuctionsCount    int64            `json:"won_auctions_count"`
	ActiveListingsCount int64            `json:"active_listings_count"`
	Warnings            []UserWarningDTO `json:"warnings"`
}
