package services

import (
	"errors"
	"time"

	"auction-backend/internal/models"
	"auction-backend/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrAuctionNotFound     = errors.New("auction not found")
	ErrUnauthorized        = errors.New("unauthorized action")
	ErrInvalidAuctionState = errors.New("invalid auction state for action")
	ErrBidTooLow           = errors.New("bid amount must be higher than current price plus increment")
)

type AuctionService struct {
	repo *repository.AuctionRepository
}

func NewAuctionService(repo *repository.AuctionRepository) *AuctionService {
	return &AuctionService{repo: repo}
}

type SellerEligibilityResult struct {
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
}

func (s *AuctionService) CheckSellerEligibility(sellerID uint) (SellerEligibilityResult, error) {
	user, err := s.repo.GetUserByID(sellerID)
	if err != nil {
		return SellerEligibilityResult{Eligible: false, Reason: "User not found"}, nil
	}
	if !user.UserType.IsSeller() && !user.UserType.IsAdmin() {
		return SellerEligibilityResult{Eligible: false, Reason: "Account level must be Seller or Admin"}, nil
	}
	return SellerEligibilityResult{Eligible: true}, nil
}

func (s *AuctionService) ListAuctions(page, limit int, status string, categoryID uint) ([]models.Auction, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	return s.repo.ListAuctions(page, limit, status, categoryID)
}

func (s *AuctionService) AdminListAuctions(page, limit int) ([]models.Auction, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	return s.repo.AdminListAuctions(page, limit)
}

func (s *AuctionService) GetDashboardStats() (map[string]interface{}, error) {
	return s.repo.GetDashboardStats()
}

func (s *AuctionService) GetAuctionDetail(id uint) (*models.Auction, error) {
	auction, err := s.repo.GetByID(id)
	if err != nil || auction == nil {
		return nil, ErrAuctionNotFound
	}
	return auction, nil
}

// UpdateStatus - Cập nhật trạng thái phiên đấu giá
func (s *AuctionService) UpdateStatus(userID uint, auctionID uint, status string) error {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return ErrAuctionNotFound
	}

	// Kiểm tra nếu không phải chủ sở hữu thì phải là Admin
	if auction.SellerID != userID {
		user, err := s.repo.GetUserByID(userID)
		if err != nil || !user.UserType.IsAdmin() {
			return ErrUnauthorized
		}
	}

	return s.repo.UpdateStatus(auctionID, status)
}

// ApproveAuction - Phê duyệt phiên đấu giá dành cho Admin
func (s *AuctionService) ApproveAuction(userID uint, auctionID uint) error {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return ErrAuctionNotFound
	}

	user, err := s.repo.GetUserByID(userID)
	if err != nil || !user.UserType.IsAdmin() {
		return ErrUnauthorized
	}

	return s.repo.UpdateStatus(auctionID, "ACTIVE")
}

type SaveDraftInput struct {
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	CategoryID    uint    `json:"category_id"`
	Location      string  `json:"location"`
	PaymentTerms  string  `json:"payment_terms"`
	ShippingPayer string  `json:"shipping_fee_payer"`
	PackageWeight float64 `json:"package_weight"`
	ReturnPolicy  string  `json:"return_policy"`
}

func (s *AuctionService) CreateDraft(sellerID uint, input SaveDraftInput) (*models.Auction, error) {
	auction := &models.Auction{
		SellerID:    sellerID,
		Title:       input.Title,
		Description: input.Description,
		CategoryID:  input.CategoryID,
		Location:    input.Location,
		Status:      models.AuctionStatusDraft,
	}

	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(auction).Error; err != nil {
			return err
		}

		if input.PaymentTerms != "" || input.ShippingPayer != "" || input.PackageWeight > 0 || input.ReturnPolicy != "" {
			sp := &models.AuctionShippingPayment{
				AuctionID:        auction.ID,
				PaymentTerms:     input.PaymentTerms,
				ShippingFeePayer: input.ShippingPayer,
				PackageWeight:    input.PackageWeight,
				ReturnPolicy:     input.ReturnPolicy,
			}
			if err := s.repo.UpsertShippingPayment(tx, sp); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.repo.GetByID(auction.ID)
}

func (s *AuctionService) UpdateDraft(sellerID uint, auctionID uint, input SaveDraftInput) (*models.Auction, error) {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return nil, ErrAuctionNotFound
	}
	if auction.SellerID != sellerID {
		return nil, ErrUnauthorized
	}
	if auction.Status != models.AuctionStatusDraft {
		return nil, ErrInvalidAuctionState
	}

	err = s.repo.DB().Transaction(func(tx *gorm.DB) error {
		auction.Title = input.Title
		auction.Description = input.Description
		auction.CategoryID = input.CategoryID
		auction.Location = input.Location

		if err := tx.Save(auction).Error; err != nil {
			return err
		}

		sp := &models.AuctionShippingPayment{
			AuctionID:        auction.ID,
			PaymentTerms:     input.PaymentTerms,
			ShippingFeePayer: input.ShippingPayer,
			PackageWeight:    input.PackageWeight,
			ReturnPolicy:     input.ReturnPolicy,
		}
		return s.repo.UpsertShippingPayment(tx, sp)
	})

	if err != nil {
		return nil, err
	}

	return s.repo.GetByID(auction.ID)
}

type PricingConfigInput struct {
	StartingBid             float64 `json:"starting_bid"`
	BidIncrement            float64 `json:"bid_increment"`
	ReservePrice            float64 `json:"reserve_price"`
	BuyNowPrice             float64 `json:"buy_now_price"`
	EstMinPrice             float64 `json:"est_min_price"`
	EstMaxPrice             float64 `json:"est_max_price"`
	AntiSnipeEnabled        bool    `json:"anti_snipe_enabled"`
	AntiSnipeTriggerMinutes int     `json:"anti_snipe_trigger_minutes"`
	AntiSnipeExtendMinutes  int     `json:"anti_snipe_extend_minutes"`
}

func (s *AuctionService) UpdatePricing(sellerID uint, auctionID uint, input PricingConfigInput) (*models.AuctionPricing, error) {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return nil, ErrAuctionNotFound
	}
	if auction.SellerID != sellerID {
		return nil, ErrUnauthorized
	}
	if auction.Status != models.AuctionStatusDraft {
		return nil, ErrInvalidAuctionState
	}

	if input.StartingBid <= 0 {
		return nil, errors.New("starting bid must be greater than 0")
	}
	if input.BidIncrement <= 0 {
		return nil, errors.New("bid increment must be greater than 0")
	}

	pricing := &models.AuctionPricing{
		AuctionID:               auction.ID,
		StartingBid:             input.StartingBid,
		BidIncrement:            input.BidIncrement,
		ReservePrice:            input.ReservePrice,
		BuyNowPrice:             input.BuyNowPrice,
		EstMinPrice:             input.EstMinPrice,
		EstMaxPrice:             input.EstMaxPrice,
		AntiSnipeEnabled:        input.AntiSnipeEnabled,
		AntiSnipeTriggerMinutes: input.AntiSnipeTriggerMinutes,
		AntiSnipeExtendMinutes:  input.AntiSnipeExtendMinutes,
	}

	if err := s.repo.UpsertPricing(nil, pricing); err != nil {
		return nil, err
	}
	return pricing, nil
}

func (s *AuctionService) GetDraftPreview(sellerID uint, auctionID uint) (*models.Auction, error) {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return nil, ErrAuctionNotFound
	}
	if auction.SellerID != sellerID {
		return nil, ErrUnauthorized
	}
	return auction, nil
}

func (s *AuctionService) PublishAuction(sellerID uint, auctionID uint, startAt, endAt time.Time) (*models.Auction, error) {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return nil, ErrAuctionNotFound
	}
	if auction.SellerID != sellerID {
		return nil, ErrUnauthorized
	}
	if auction.Status != models.AuctionStatusDraft {
		return nil, errors.New("only draft auctions can be published")
	}

	now := time.Now()
	if !endAt.After(startAt) {
		return nil, errors.New("end time must be after start time")
	}
	if endAt.Before(now) {
		return nil, errors.New("end time cannot be in the past")
	}

	if auction.Pricing == nil || auction.Pricing.StartingBid <= 0 {
		return nil, errors.New("auction pricing must be configured before publishing")
	}

	if startAt.Before(now) || startAt.Equal(now) {
		auction.Status = models.AuctionStatusLive
	} else {
		auction.Status = models.AuctionStatusScheduled
	}

	auction.StartAt = &startAt
	auction.EndAt = &endAt

	if err := s.repo.UpdateAuction(auction); err != nil {
		return nil, err
	}
	return auction, nil
}

func (s *AuctionService) ProcessBid(auctionID uint, bidderID uint, amount float64) (*models.Bid, bool, *time.Time, error) {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return nil, false, nil, ErrAuctionNotFound
	}
	if auction.Status != models.AuctionStatusLive {
		return nil, false, nil, errors.New("auction is not live")
	}

	highestBid, err := s.repo.GetHighestBid(auctionID)
	if err != nil {
		return nil, false, nil, err
	}

	var minAllowed float64
	bidType := models.BidTypeCompetingBid

	if highestBid == nil {
		if auction.Pricing != nil {
			minAllowed = auction.Pricing.StartingBid
		}
		bidType = models.BidTypeStartingBid
	} else {
		minAllowed = highestBid.Amount
		if auction.Pricing != nil {
			minAllowed += auction.Pricing.BidIncrement
		}
	}

	if amount < minAllowed {
		return nil, false, nil, ErrBidTooLow
	}

	bid := &models.Bid{
		AuctionID: auctionID,
		BidderID:  bidderID,
		Amount:    amount,
		BidType:   bidType,
	}

	if err := s.repo.CreateBid(bid); err != nil {
		return nil, false, nil, err
	}

	var extended bool
	var newEndAt *time.Time

	if auction.Pricing != nil && auction.Pricing.AntiSnipeEnabled && auction.EndAt != nil {
		triggerWindow := time.Duration(auction.Pricing.AntiSnipeTriggerMinutes) * time.Minute
		extendDuration := time.Duration(auction.Pricing.AntiSnipeExtendMinutes) * time.Minute

		if time.Until(*auction.EndAt) <= triggerWindow {
			updatedTime := auction.EndAt.Add(extendDuration)
			auction.EndAt = &updatedTime
			_ = s.repo.UpdateAuction(auction)
			extended = true
			newEndAt = &updatedTime
		}
	}

	return bid, extended, newEndAt, nil
}

func (s *AuctionService) DeleteAuction(sellerID uint, auctionID uint) error {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return ErrAuctionNotFound
	}

	if auction.SellerID != sellerID {
		user, err := s.repo.GetUserByID(sellerID)
		if err != nil || !user.UserType.IsAdmin() {
			return ErrUnauthorized
		}
	}

	if auction.Status != models.AuctionStatusDraft && auction.Status != models.AuctionStatusScheduled {
		return errors.New("không thể xóa phiên đấu giá đang diễn ra hoặc đã kết thúc")
	}

	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("auction_id = ?", auctionID).Delete(&models.AuctionPricing{}).Error; err != nil {
			return err
		}
		if err := tx.Where("auction_id = ?", auctionID).Delete(&models.AuctionShippingPayment{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Auction{}, auctionID).Error; err != nil {
			return err
		}
		return nil
	})
}