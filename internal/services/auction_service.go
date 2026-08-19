package services

import (
	"errors"
	"time"

	"auction-backend/internal/models"
	"auction-backend/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrAuctionNotFound     = errors.New("không tìm thấy phiên đấu giá")
	ErrUnauthorized        = errors.New("không có quyền thực hiện thao tác này")
	ErrInvalidAuctionState = errors.New("trạng thái phiên đấu giá không hợp lệ")
	ErrBidTooLow           = errors.New("số tiền đặt giá phải cao hơn giá hiện tại cộng với bước giá")
	ErrSelfBidding         = errors.New("chủ phiên đấu giá không được phép tự đặt giá sản phẩm của mình")
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

func (s *AuctionService) ListSellerAuctions(sellerID uint, status string, page, limit int) ([]models.Auction, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListBySeller(sellerID, page, limit, status)
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

func (s *AuctionService) UpdateStatus(userID uint, auctionID uint, status string) error {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return ErrAuctionNotFound
	}

	if auction.SellerID != userID {
		user, err := s.repo.GetUserByID(userID)
		if err != nil || !user.UserType.IsAdmin() {
			return ErrUnauthorized
		}
	}

	return s.repo.UpdateStatus(auctionID, status)
}

func (s *AuctionService) PauseAuction(adminID uint, auctionID uint) error {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return ErrAuctionNotFound
	}

	user, err := s.repo.GetUserByID(adminID)
	if err != nil || !user.UserType.IsAdmin() {
		return ErrUnauthorized
	}

	if auction.Status != models.AuctionStatusLive && string(auction.Status) != "ACTIVE" {
		return errors.New("chỉ có thể tạm dừng phiên đang diễn ra (LIVE / ACTIVE)")
	}

	return s.repo.UpdateStatus(auctionID, string(models.AuctionStatusPaused))
}

func (s *AuctionService) ResumeAuction(adminID uint, auctionID uint) error {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return ErrAuctionNotFound
	}

	user, err := s.repo.GetUserByID(adminID)
	if err != nil || !user.UserType.IsAdmin() {
		return ErrUnauthorized
	}

	if auction.Status != models.AuctionStatusPaused {
		return errors.New("chỉ có thể tiếp tục phiên đang bị tạm dừng (PAUSED)")
	}

	return s.repo.UpdateStatus(auctionID, string(models.AuctionStatusLive))
}

func (s *AuctionService) CancelAuction(adminID uint, auctionID uint, reason string) error {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return ErrAuctionNotFound
	}

	user, err := s.repo.GetUserByID(adminID)
	if err != nil || !user.UserType.IsAdmin() {
		return ErrUnauthorized
	}

	if auction.Status == models.AuctionStatusEnded || auction.Status == models.AuctionStatusCancelled {
		return errors.New("phiên đấu giá đã kết thúc hoặc đã hủy từ trước")
	}

	auction.Status = models.AuctionStatusCancelled
	auction.RejectionReason = reason
	return s.repo.UpdateAuction(auction)
}

func (s *AuctionService) ApproveAuction(userID uint, auctionID uint) error {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return ErrAuctionNotFound
	}

	user, err := s.repo.GetUserByID(userID)
	if err != nil || !user.UserType.IsAdmin() {
		return ErrUnauthorized
	}

	if auction.Status != models.AuctionStatusPendingApproval {
		return ErrInvalidAuctionState
	}
	if auction.StartAt == nil || auction.EndAt == nil || !auction.EndAt.After(*auction.StartAt) {
		return errors.New("phiên đấu giá phải có thời gian bắt đầu/kết thúc hợp lệ trước khi phê duyệt")
	}
	if !auction.EndAt.After(time.Now()) {
		return errors.New("thời gian kết thúc phải nằm trong tương lai")
	}
	if auction.StartAt.After(time.Now()) {
		auction.Status = models.AuctionStatusScheduled
	} else {
		auction.Status = models.AuctionStatusLive
	}
	auction.RejectionReason = ""
	return s.repo.UpdateAuction(auction)
}

func (s *AuctionService) RejectAuction(userID, auctionID uint, reason string) error {
	auction, err := s.repo.GetByID(auctionID)
	if err != nil || auction == nil {
		return ErrAuctionNotFound
	}
	user, err := s.repo.GetUserByID(userID)
	if err != nil || !user.UserType.IsAdmin() {
		return ErrUnauthorized
	}
	if auction.Status != models.AuctionStatusPendingApproval {
		return ErrInvalidAuctionState
	}
	auction.Status, auction.RejectionReason = models.AuctionStatusRejected, reason
	return s.repo.UpdateAuction(auction)
}

func (s *AuctionService) RelistAuction(sellerID, auctionID uint, startAt, endAt time.Time) (*models.Auction, error) {
	if !endAt.After(startAt) {
		return nil, errors.New("thời gian kết thúc phải sau thời gian bắt đầu")
	}
	if !startAt.After(time.Now()) {
		return nil, errors.New("thời gian bắt đầu phải ở tương lai")
	}
	var result *models.Auction
	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		auction, err := s.repo.GetByIDForUpdate(tx, auctionID)
		if err != nil || auction == nil {
			return ErrAuctionNotFound
		}
		if auction.SellerID != sellerID {
			return ErrUnauthorized
		}
		// Cho phép Relist khi phiên ở trạng thái ENDED, CANCELLED hoặc FAILED
		if auction.Status != models.AuctionStatusEnded &&
			auction.Status != models.AuctionStatusCancelled &&
			auction.Status != models.AuctionStatusFailed {
			return ErrInvalidAuctionState
		}
		if err := tx.Where("auction_id = ?", auctionID).Delete(&models.Bid{}).Error; err != nil {
			return err
		}
		auction.StartAt, auction.EndAt = &startAt, &endAt
		auction.Status, auction.SaleStatus = models.AuctionStatusPendingApproval, "PENDING"
		auction.WinnerID, auction.WinningBidID, auction.WinningAmount = nil, nil, nil
		auction.RejectionReason = ""
		if err := s.repo.UpdateAuctionInTx(tx, auction); err != nil {
			return err
		}
		result = auction
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CloseExpiredAuctions tự động đóng các phiên đấu giá đã quá hạn (Dùng cho Cron Job hoặc Worker)
func (s *AuctionService) CloseExpiredAuctions() error {
	now := time.Now()

	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		var expiredAuctions []models.Auction

		// Tìm các phiên đấu giá đang LIVE/ACTIVE nhưng đã quá EndAt
		err := tx.Where("status IN ? AND end_at IS NOT NULL AND end_at <= ?", []interface{}{models.AuctionStatusLive, "ACTIVE"}, now).
			Find(&expiredAuctions).Error
		if err != nil {
			return err
		}

		for _, auction := range expiredAuctions {
			highestBid, err := s.repo.GetHighestBidInTx(tx, auction.ID)
			if err != nil {
				continue
			}

			if highestBid == nil {
				// Không có lượt đặt giá -> Phiên đấu giá thất bại (FAILED)
				auction.Status = models.AuctionStatusFailed
				auction.SaleStatus = "UNSOLD"
			} else {
				// Có người đặt giá -> Đấu giá thành công (ENDED)
				auction.Status = models.AuctionStatusEnded
				auction.SaleStatus = "SOLD"
				auction.WinnerID = &highestBid.BidderID
				auction.WinningBidID = &highestBid.ID
				auction.WinningAmount = &highestBid.Amount
			}

			if err := s.repo.UpdateAuctionInTx(tx, &auction); err != nil {
				return err
			}
		}

		return nil
	})
}

// XỬ LÝ ĐẶT GIÁ AN TOÀN (TRANSACTION + PESSIMISTIC LOCKING + ANTI-SNIPE)
func (s *AuctionService) ProcessBid(auctionID uint, bidderID uint, amount float64) (*models.Bid, bool, *time.Time, error) {
	var (
		resultBid *models.Bid
		extended  bool
		newEndAt  *time.Time
	)

	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		// 1. Lock dòng record phiên đấu giá trong DB (Pessimistic Locking FOR UPDATE)
		auction, err := s.repo.GetByIDForUpdate(tx, auctionID)
		if err != nil || auction == nil {
			return ErrAuctionNotFound
		}

		// 2. Chặn chủ hàng tự bid sản phẩm của chính mình
		if auction.SellerID == bidderID {
			return ErrSelfBidding
		}

		// 3. Kiểm tra trạng thái (Hỗ trợ cả 'LIVE' lẫn 'ACTIVE')
		if auction.Status != models.AuctionStatusLive && string(auction.Status) != "ACTIVE" {
			return errors.New("phiên đấu giá hiện không ở trạng thái LIVE hoặc ACTIVE")
		}

		now := time.Now()
		if auction.StartAt != nil && now.Before(*auction.StartAt) {
			return errors.New("phiên đấu giá chưa đến thời gian bắt đầu")
		}
		if auction.EndAt != nil && !now.Before(*auction.EndAt) {
			return errors.New("phiên đấu giá đã kết thúc")
		}

		// 4. Lấy lượt bid cao nhất hiện tại trong Transaction
		highestBid, err := s.repo.GetHighestBidInTx(tx, auctionID)
		if err != nil {
			return err
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
			return ErrBidTooLow
		}

		// 5. Tạo lượt bid mới
		bid := &models.Bid{
			AuctionID: auctionID,
			BidderID:  bidderID,
			Amount:    amount,
			BidType:   bidType,
		}
		if err := s.repo.CreateBidInTx(tx, bid); err != nil {
			return err
		}

		// 6. Cập nhật CurrentPrice trực tiếp vào bảng auctions
		auction.CurrentPrice = amount

		// 7. Xử lý Kích hoạt Anti-Snipe (Gia hạn thời gian)
		if auction.Pricing != nil && auction.Pricing.AntiSnipeEnabled && auction.EndAt != nil {
			triggerWindow := time.Duration(auction.Pricing.AntiSnipeTriggerMinutes) * time.Minute
			extendDuration := time.Duration(auction.Pricing.AntiSnipeExtendMinutes) * time.Minute

			if now.Add(triggerWindow).Compare(*auction.EndAt) >= 0 {
				updatedTime := auction.EndAt.Add(extendDuration)
				auction.EndAt = &updatedTime
				extended = true
				newEndAt = &updatedTime
			}
		}

		if err := s.repo.UpdateAuctionInTx(tx, auction); err != nil {
			return err
		}

		resultBid = bid
		return nil
	})

	if err != nil {
		return nil, false, nil, err
	}

	return resultBid, extended, newEndAt, nil
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

	StartingBid float64 `json:"starting_bid"`
	StartPrice  float64 `json:"start_price"`
	Price       float64 `json:"price"`

	BidIncrement float64 `json:"bid_increment"`
	BidStep      float64 `json:"bid_step"`

	BuyNowPrice  float64 `json:"buy_now_price"`
	ReservePrice float64 `json:"reserve_price"`
}

func (input *SaveDraftInput) GetStartingBid() float64 {
	if input.StartingBid > 0 {
		return input.StartingBid
	}
	if input.StartPrice > 0 {
		return input.StartPrice
	}
	return input.Price
}

func (input *SaveDraftInput) GetBidIncrement() float64 {
	if input.BidIncrement > 0 {
		return input.BidIncrement
	}
	return input.BidStep
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

		startBid := input.GetStartingBid()
		bidInc := input.GetBidIncrement()
		if startBid > 0 {
			pricing := &models.AuctionPricing{
				AuctionID:    auction.ID,
				StartingBid:  startBid,
				BidIncrement: bidInc,
				BuyNowPrice:  input.BuyNowPrice,
				ReservePrice: input.ReservePrice,
			}
			if err := s.repo.UpsertPricing(tx, pricing); err != nil {
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
		if err := s.repo.UpsertShippingPayment(tx, sp); err != nil {
			return err
		}

		startBid := input.GetStartingBid()
		bidInc := input.GetBidIncrement()
		if startBid > 0 {
			pricing := &models.AuctionPricing{
				AuctionID:    auction.ID,
				StartingBid:  startBid,
				BidIncrement: bidInc,
				BuyNowPrice:  input.BuyNowPrice,
				ReservePrice: input.ReservePrice,
			}
			if err := s.repo.UpsertPricing(tx, pricing); err != nil {
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

	auction.Status = models.AuctionStatusPendingApproval
	auction.StartAt = &startAt
	auction.EndAt = &endAt

	if err := s.repo.UpdateAuction(auction); err != nil {
		return nil, err
	}
	return auction, nil
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
