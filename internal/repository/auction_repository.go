package repository

import (
	"errors"

	"auction-backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuctionRepository struct {
	db *gorm.DB
}

func NewAuctionRepository(db *gorm.DB) *AuctionRepository {
	return &AuctionRepository{db: db}
}

func (r *AuctionRepository) DB() *gorm.DB {
	return r.db
}

func (r *AuctionRepository) CreateAuction(auction *models.Auction) error {
	return r.db.Create(auction).Error
}

func (r *AuctionRepository) GetByID(id uint) (*models.Auction, error) {
	var auction models.Auction
	err := r.db.Preload("Pricing").
		Preload("ShippingPayment").
		Preload("Seller").
		First(&auction, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &auction, nil
}

// GetByIDForUpdate locks the auction row until the surrounding transaction ends.
// Bids for the same auction must acquire this lock before calculating a minimum bid.
func (r *AuctionRepository) GetByIDForUpdate(tx *gorm.DB, id uint) (*models.Auction, error) {
	var auction models.Auction
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("Pricing").
		Preload("ShippingPayment").
		Preload("Seller").
		First(&auction, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &auction, nil
}

func (r *AuctionRepository) UpdateAuction(auction *models.Auction) error {
	return r.db.Save(auction).Error
}

func (r *AuctionRepository) UpdateAuctionInTx(tx *gorm.DB, auction *models.Auction) error {
	return tx.Save(auction).Error
}

// UpdateStatus - Cập nhật trực tiếp cột status của auction theo ID
func (r *AuctionRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.Auction{}).Where("id = ?", id).Update("status", status).Error
}

// ListAuctions lấy danh sách phiên đấu giá public
func (r *AuctionRepository) ListAuctions(page, limit int, status string, categoryID uint) ([]models.Auction, int64, error) {
	var auctions []models.Auction
	var total int64

	query := r.db.Model(&models.Auction{})
	if status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status IN ?", []models.AuctionStatus{models.AuctionStatusLive, models.AuctionStatusScheduled})
	}

	if categoryID > 0 {
		query = query.Where("category_id = ?", categoryID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	err := query.Preload("Pricing").
		Preload("ShippingPayment").
		Preload("Seller").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&auctions).Error

	return auctions, total, err
}

// AdminListAuctions lấy toàn bộ danh sách cho Admin quản lý
func (r *AuctionRepository) AdminListAuctions(page, limit int) ([]models.Auction, int64, error) {
	var auctions []models.Auction
	var total int64

	query := r.db.Model(&models.Auction{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	err := query.Preload("Pricing").
		Preload("ShippingPayment").
		Preload("Seller").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&auctions).Error

	return auctions, total, err
}

func (r *AuctionRepository) ListBySeller(sellerID uint, page, limit int, status string) ([]models.Auction, int64, error) {
	var auctions []models.Auction
	var total int64
	query := r.db.Model(&models.Auction{}).Where("seller_id = ?", sellerID)
	if status != "" { query = query.Where("status = ?", status) }
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Preload("Pricing").Preload("ShippingPayment").Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&auctions).Error
	return auctions, total, err
}

func (r *AuctionRepository) ListWonByUser(userID uint) ([]models.Auction, error) {
	var auctions []models.Auction
	err := r.db.Where("winner_id = ? AND sale_status = ?", userID, "SOLD").Preload("Pricing").Order("updated_at DESC").Find(&auctions).Error
	return auctions, err
}

func (r *AuctionRepository) ListBidParticipations(userID uint) ([]models.Auction, error) {
	var auctions []models.Auction
	err := r.db.Model(&models.Auction{}).Distinct("auctions.*").Joins("JOIN bids ON bids.auction_id = auctions.id").Where("bids.bidder_id = ?", userID).Preload("Pricing").Order("auctions.updated_at DESC").Find(&auctions).Error
	return auctions, err
}

func (r *AuctionRepository) CountSellerAuctions(userID uint) (int64, error) { var count int64; err := r.db.Model(&models.Auction{}).Where("seller_id = ?", userID).Count(&count).Error; return count, err }

// GetDashboardStats Thống kê dữ liệu tổng quan cho Admin Dashboard
func (r *AuctionRepository) GetDashboardStats() (map[string]interface{}, error) {
	var totalAuctions int64
	var liveAuctions int64
	var totalBids int64

	r.db.Model(&models.Auction{}).Count(&totalAuctions)
	r.db.Model(&models.Auction{}).Where("status = ?", models.AuctionStatusLive).Count(&liveAuctions)
	r.db.Model(&models.Bid{}).Count(&totalBids)

	stats := map[string]interface{}{
		"total_auctions": totalAuctions,
		"live_auctions":  liveAuctions,
		"total_bids":     totalBids,
	}
	return stats, nil
}

func (r *AuctionRepository) UpsertPricing(tx *gorm.DB, pricing *models.AuctionPricing) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	var existing models.AuctionPricing
	err := db.Where("auction_id = ?", pricing.AuctionID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(pricing).Error
	} else if err != nil {
		return err
	}
	pricing.ID = existing.ID
	return db.Save(pricing).Error
}

func (r *AuctionRepository) UpsertShippingPayment(tx *gorm.DB, sp *models.AuctionShippingPayment) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	var existing models.AuctionShippingPayment
	err := db.Where("auction_id = ?", sp.AuctionID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(sp).Error
	} else if err != nil {
		return err
	}
	sp.ID = existing.ID
	return db.Save(sp).Error
}

func (r *AuctionRepository) CreateBid(bid *models.Bid) error {
	return r.db.Create(bid).Error
}

func (r *AuctionRepository) CreateBidInTx(tx *gorm.DB, bid *models.Bid) error {
	return tx.Create(bid).Error
}

func (r *AuctionRepository) GetHighestBid(auctionID uint) (*models.Bid, error) {
	var bid models.Bid
	err := r.db.Where("auction_id = ?", auctionID).
		Order("amount DESC, created_at ASC").
		First(&bid).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &bid, err
}

func (r *AuctionRepository) GetHighestBidInTx(tx *gorm.DB, auctionID uint) (*models.Bid, error) {
	var bid models.Bid
	err := tx.Where("auction_id = ?", auctionID).
		Order("amount DESC, created_at ASC").
		First(&bid).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &bid, err
}

func (r *AuctionRepository) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("user not found")
	}
	return &user, err
}
