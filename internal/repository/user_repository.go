package repository

import (
	"time"

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

func (r *UserRepository) ListUsersFiltered(registered24h bool, isSuspicious bool, search string) ([]models.User, error) {
	var users []models.User
	query := r.db.Model(&models.User{})

	if registered24h {
		since := time.Now().Add(-24 * time.Hour)
		query = query.Where("created_at >= ?", since)
	}

	if isSuspicious {
		query = query.Where("is_suspicious = ?", true)
	}

	if search != "" {
		likeTerm := "%" + search + "%"
		query = query.Where("full_name LIKE ? OR email LIKE ?", likeTerm, likeTerm)
	}

	err := query.Order("id DESC").Find(&users).Error
	return users, err
}

func (r *UserRepository) GetUserSummary(userID uint) (*models.UserSummaryDTO, error) {
	var user models.User
	if err := r.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	var totalSpent float64
	_ = r.db.Table("auctions").
		Where("winner_id = ? AND status IN ('SUCCESS', 'ENDED', 'completed')", userID).
		Select("COALESCE(SUM(current_price), 0)").
		Scan(&totalSpent)

	var wonCount int64
	_ = r.db.Table("auctions").
		Where("winner_id = ?", userID).
		Count(&wonCount)

	var activeListings int64
	_ = r.db.Table("auctions").
		Where("seller_id = ? AND status IN ('ACTIVE', 'active')", userID).
		Count(&activeListings)

	var warnings []models.UserWarning
	_ = r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&warnings)

	warningDTOs := make([]models.UserWarningDTO, 0, len(warnings))
	for _, w := range warnings {
		warningDTOs = append(warningDTOs, models.UserWarningDTO{
			ID:        w.ID,
			Reason:    w.Reason,
			CreatedAt: w.CreatedAt,
		})
	}

	return &models.UserSummaryDTO{
		ID:                  user.ID,
		FullName:            user.FullName,
		Email:               user.Email,
		UserType:            user.UserType,
		IsSuspicious:        user.IsSuspicious,
		TotalSpent:          totalSpent,
		WonAuctionsCount:    wonCount,
		ActiveListingsCount: activeListings,
		Warnings:            warningDTOs,
	}, nil
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

func (r *UserRepository) GetUserBids(userID uint) ([]models.UserBidItemDTO, error) {
	var results []models.UserBidItemDTO

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

func (r *UserRepository) GetWonAuctions(userID uint) ([]models.Auction, error) {
	var auctions []models.Auction
	err := r.db.Where("winner_id = ?", userID).
		Preload("Pricing").
		Preload("ShippingPayment").
		Order("updated_at DESC").
		Find(&auctions).Error
	return auctions, err
}

func (r *UserRepository) GetProfileStats(userID uint) (participatingCount int64, wonCount int64, myAuctionCount int64, err error) {
	err = r.db.Table("bids").Where("bidder_id = ?", userID).Select("COUNT(DISTINCT auction_id)").Count(&participatingCount).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.Auction{}).Where("winner_id = ?", userID).Count(&wonCount).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.Auction{}).Where("seller_id = ?", userID).Count(&myAuctionCount).Error
	return
}
