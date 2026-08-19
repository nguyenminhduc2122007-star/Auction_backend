package models

import (
	"time"
)

type AuctionStatus string

const (
	AuctionStatusDraft           AuctionStatus = "DRAFT"
	AuctionStatusPendingApproval AuctionStatus = "PENDING_APPROVAL"
	AuctionStatusRejected        AuctionStatus = "REJECTED"
	AuctionStatusScheduled       AuctionStatus = "SCHEDULED"
	AuctionStatusLive            AuctionStatus = "LIVE"
	AuctionStatusPaused          AuctionStatus = "PAUSED"
	AuctionStatusEnded           AuctionStatus = "ENDED"
	AuctionStatusCancelled       AuctionStatus = "CANCELLED"
	AuctionStatusFailed          AuctionStatus = "FAILED" // 🟢 Đã bổ sung hằng số này
)

type BidType string

const (
	BidTypeStartingBid  BidType = "STARTING_BID"
	BidTypeCompetingBid BidType = "COMPETING_BID"
)

// Auction Model chính
type Auction struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	SellerID    uint          `gorm:"not null;index" json:"seller_id"`
	Seller      *User         `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
	Title       string        `gorm:"type:varchar(255);not null" json:"title"`
	Description string        `gorm:"type:text" json:"description"`
	CategoryID  uint          `gorm:"not null;index" json:"category_id"`
	Location    string        `gorm:"type:varchar(255)" json:"location"`
	ImageURL    string        `gorm:"type:text" json:"image_url,omitempty"`
	Status      AuctionStatus `gorm:"type:varchar(32);default:'DRAFT';index" json:"status"`

	StartAt *time.Time `json:"start_at,omitempty"`
	EndAt   *time.Time `json:"end_at,omitempty"`

	WinnerID        *uint    `json:"winner_id,omitempty"`
	WinningBidID    *uint    `json:"winning_bid_id,omitempty"`
	WinningAmount   *float64 `json:"winning_amount,omitempty"`
	CurrentPrice    float64  `gorm:"type:numeric(15,2);default:0" json:"current_price"`
	SaleStatus      string   `gorm:"type:varchar(20);default:'PENDING'" json:"sale_status"`
	RejectionReason string   `gorm:"type:text" json:"rejection_reason,omitempty"`

	// Relationships
	Pricing         *AuctionPricing         `gorm:"foreignKey:AuctionID;constraint:OnDelete:CASCADE" json:"pricing,omitempty"`
	ShippingPayment *AuctionShippingPayment `gorm:"foreignKey:AuctionID;constraint:OnDelete:CASCADE" json:"shipping_payment,omitempty"`
	Bids            []Bid                   `gorm:"foreignKey:AuctionID" json:"bids,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Bảng Cấu hình Giá & Anti-Snipe
type AuctionPricing struct {
	ID                      uint    `gorm:"primaryKey" json:"id"`
	AuctionID               uint    `gorm:"uniqueIndex;not null" json:"auction_id"`
	StartingBid             float64 `gorm:"type:numeric(15,2);not null" json:"starting_bid"`
	BidIncrement            float64 `gorm:"type:numeric(15,2);not null" json:"bid_increment"`
	ReservePrice            float64 `gorm:"type:numeric(15,2);default:0" json:"reserve_price"`
	BuyNowPrice             float64 `gorm:"type:numeric(15,2);default:0" json:"buy_now_price"`
	EstMinPrice             float64 `gorm:"type:numeric(15,2);default:0" json:"est_min_price"`
	EstMaxPrice             float64 `gorm:"type:numeric(15,2);default:0" json:"est_max_price"`
	AntiSnipeEnabled        bool    `gorm:"default:true" json:"anti_snipe_enabled"`
	AntiSnipeTriggerMinutes int     `gorm:"default:2" json:"anti_snipe_trigger_minutes"`
	AntiSnipeExtendMinutes  int     `gorm:"default:2" json:"anti_snipe_extend_minutes"`
}

// Bảng Vận chuyển & Thanh toán
type AuctionShippingPayment struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	AuctionID        uint    `gorm:"uniqueIndex;not null" json:"auction_id"`
	PaymentTerms     string  `gorm:"type:varchar(255)" json:"payment_terms"`
	ShippingFeePayer string  `gorm:"type:varchar(64)" json:"shipping_fee_payer"`
	PackageWeight    float64 `gorm:"type:numeric(10,2)" json:"package_weight"`
	ReturnPolicy     string  `gorm:"type:varchar(255)" json:"return_policy"`
}

// 🟢 Helper Methods hỗ trợ Kiểm Tra Trạng Thái
func (a *Auction) IsLive() bool {
	return a.Status == AuctionStatusLive
}

func (a *Auction) IsPaused() bool {
	return a.Status == AuctionStatusPaused
}

func (a *Auction) IsFailed() bool {
	return a.Status == AuctionStatusFailed
}

func (a *Auction) IsEnded() bool {
	return a.Status == AuctionStatusEnded
}

func (a *Auction) CanBeEdited() bool {
	return a.Status == AuctionStatusDraft || a.Status == AuctionStatusPendingApproval || a.Status == AuctionStatusRejected
}

func (a *Auction) CanBeRelisted() bool {
	return a.Status == AuctionStatusEnded || a.Status == AuctionStatusCancelled || a.Status == AuctionStatusFailed
}

// 🟢 DTOs Structs Phục Vụ REST APIs
type CreateAuctionInput struct {
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	CategoryID  uint      `json:"category_id" binding:"required"`
	SellerID    uint      `json:"seller_id" binding:"required"`
	Location    string    `json:"location"`
	ImageURL    string    `json:"image_url"`
	StartingBid float64   `json:"starting_bid" binding:"required,gt=0"`
	BidStep     float64   `json:"bid_step" binding:"required,gt=0"`
	StartAt     time.Time `json:"start_at" binding:"required"`
	EndAt       time.Time `json:"end_at" binding:"required"`
}

type UpdateAuctionInput struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	CategoryID  *uint      `json:"category_id"`
	Location    *string    `json:"location"`
	ImageURL    *string    `json:"image_url"`
	StartingBid *float64   `json:"starting_bid"`
	BidStep     *float64   `json:"bid_step"`
	StartAt     *time.Time `json:"start_at"`
	EndAt       *time.Time `json:"end_at"`
}

type ChangeAuctionStatusInput struct {
	Action string `json:"action" binding:"required,oneof=approve pause resume cancel"`
}

type ExtendAuctionTimeInput struct {
	AddedMinutes int `json:"added_minutes" binding:"required,gt=0,lte=60"`
}

type PublishAuctionInput struct {
	StartAt time.Time `json:"start_at" binding:"required"`
	EndAt   time.Time `json:"end_at" binding:"required"`
}
