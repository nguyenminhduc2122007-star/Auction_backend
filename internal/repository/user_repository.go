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

func (r *UserRepository) UpdateProfile(user *models.User) error {
	return r.db.Save(user).Error
}

// --- F-005 SPECIFIC METHODS ---

// GetUserBids - Lấy danh sách các phiên đấu giá người dùng đang tham gia bid
func (r *UserRepository) GetUserBids(userID uint) ([]models.UserBidItemDTO, error) {
	var results []models.UserBidItemDTO

	// Subquery tìm max bid của user cho mỗi auction
	query := `
		SELECT 
			a.id AS auction_id,
			a.title,
			COALESCE((SELECT MAX(amount) FROM bids WHERE auction_id = a.id), 0) AS current_price,
			ub.user_max_bid,
			(ub.user_max_bid >= COALESCE((SELECT MAX(amount) FROM bids WHERE auction_id = a.id), 0)) AS is_winning,
			a.status
		FROM (
			SELECT auction_id, MAX(amount) AS user_max_bid
			FROM bids
			WHERE bidder_id = ?
			GROUP BY auction_id
		) ub
		JOIN auctions a ON a.id = ub.auction_id
		ORDER BY a.updated_at DESC
	`

	err := r.db.Raw(query, userID).Scan(&results).Error
	return results, err
}

// GetWonAuctions - Lấy danh sách các phiên người dùng đã thắng
func (r *UserRepository) GetWonAuctions(userID uint) ([]models.Auction, error) {
	var auctions []models.Auction
	err := r.db.Where("winner_id = ?", userID).
		Preload("Pricing").
		Preload("ShippingPayment").
		Order("updated_at DESC").
		Find(&auctions).Error
	return auctions, err
}

// GetProfileStats - Lấy thông tin thống kê nhanh cho Header Profile
func (r *UserRepository) GetProfileStats(userID uint) (participatingCount int64, wonCount int64, myAuctionCount int64, err error) {
	// 1. Số phiên đang tham gia
	err = r.db.Table("bids").Where("bidder_id = ?", userID).Select("COUNT(DISTINCT auction_id)").Count(&participatingCount).Error
	if err != nil {
		return
	}

	// 2. Số phiên đã thắng
	err = r.db.Model(&models.Auction{}).Where("winner_id = ?", userID).Count(&wonCount).Error
	if err != nil {
		return
	}

	// 3. Số phiên đã/đang đăng bán
	err = r.db.Model(&models.Auction{}).Where("seller_id = ?", userID).Count(&myAuctionCount).Error
	return
}