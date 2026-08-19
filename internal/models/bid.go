package models

import (
	"time"
)

// Bảng Lịch sử Lượt ra giá (Bid) trong DB
type Bid struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AuctionID uint      `gorm:"not null;index:idx_auction_created,priority:1" json:"auction_id"`
	BidderID  uint      `gorm:"not null;index" json:"bidder_id"`
	Bidder    *User     `gorm:"foreignKey:BidderID" json:"bidder,omitempty"`
	Amount    float64   `gorm:"type:numeric(15,2);not null" json:"amount"`
	BidType   BidType   `gorm:"type:varchar(32);not null" json:"bid_type"`
	CreatedAt time.Time `gorm:"index:idx_auction_created,priority:2" json:"created_at"`
}

// DTO Input nhận dữ liệu đặt giá từ Client (HTTP API)
type PlaceBidInput struct {
	AuctionID uint    `json:"auction_id" binding:"required"`
	BidAmount float64 `json:"bid_amount" binding:"required,gt=0"`
}

// 🟢 DTO Output phát Real-time WebSocket cho Client (Task 5 & Task 6)
type BidUpdateDTO struct {
	ID           uint      `json:"id"`
	AuctionID    uint      `json:"auction_id"`
	BidderName   string    `json:"bidder_name"` // Tên hiển thị người đặt
	Amount       float64   `json:"amount"`      // Giá đặt lượt này
	BidType      BidType   `json:"bid_type"`
	CurrentPrice float64   `json:"current_price"` // Giá hiện tại mới nhất của phiên
	IsHighest    bool      `json:"is_highest"`    // Báo hiệu đây là giá cao nhất hiện tại
	CreatedAt    time.Time `json:"created_at"`
}
