package worker

import (
	"log"
	"time"

	"auction-backend/internal/handlers"
	"auction-backend/internal/models"
	"auction-backend/internal/repository"

	"gorm.io/gorm"
)

type AuctionWorker struct {
	db       *gorm.DB
	repo     *repository.AuctionRepository
	stopChan chan struct{}
}

func NewAuctionWorker(db *gorm.DB) *AuctionWorker {
	return &AuctionWorker{
		db:       db,
		repo:     repository.NewAuctionRepository(db),
		stopChan: make(chan struct{}),
	}
}

func (w *AuctionWorker) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				w.processAuctions()
			case <-w.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
	log.Printf("Auction Background Worker started (interval: %v)", interval)
}

func (w *AuctionWorker) Stop() {
	close(w.stopChan)
}

func (w *AuctionWorker) processAuctions() {
	now := time.Now()

	// 1. Tự động chuyển SCHEDULED -> LIVE khi đến giờ start_at
	resStart := w.db.Model(&models.Auction{}).
		Where("status = ? AND start_at <= ?", models.AuctionStatusScheduled, now).
		Update("status", models.AuctionStatusLive)
	if resStart.Error != nil {
		log.Printf("[Worker Error] Failed to start scheduled auctions: %v", resStart.Error)
	}

	// 2. Tìm các phiên LIVE đã quá giờ end_at để kết thúc
	var endingAuctions []models.Auction
	if err := w.db.Where("status = ? AND end_at <= ?", models.AuctionStatusLive, now).
		Find(&endingAuctions).Error; err != nil {
		log.Printf("[Worker Error] Failed to query ending auctions: %v", err)
		return
	}

	for _, auction := range endingAuctions {
		var endedAuction *models.Auction
		err := w.db.Transaction(func(tx *gorm.DB) error {
			// Serialize completion with ProcessBid and re-check eligibility after
			// acquiring the row lock.
			lockedAuction, err := w.repo.GetByIDForUpdate(tx, auction.ID)
			if err != nil || lockedAuction == nil {
				return err
			}
			if lockedAuction.Status != models.AuctionStatusLive || lockedAuction.EndAt == nil || lockedAuction.EndAt.After(now) {
				return nil
			}

			highestBid, err := w.repo.GetHighestBidInTx(tx, lockedAuction.ID)
			if err != nil {
				return err
			}

			lockedAuction.Status = models.AuctionStatusEnded
			if highestBid == nil {
				lockedAuction.SaleStatus = "UNSOLD"
			} else {
				winningAmount := highestBid.Amount
				lockedAuction.SaleStatus = "SOLD"
				lockedAuction.WinnerID = &highestBid.BidderID
				lockedAuction.WinningBidID = &highestBid.ID
				lockedAuction.WinningAmount = &winningAmount
			}

			if err := w.repo.UpdateAuctionInTx(tx, lockedAuction); err != nil {
				return err
			}
			endedAuction = lockedAuction
			return nil
		})
		if err != nil {
			log.Printf("[Worker Error] Failed to end auction ID %d: %v", auction.ID, err)
			continue
		}
		if endedAuction == nil {
			continue
		}

		log.Printf("[Worker] Auction ID %d ended successfully with sale status %s", endedAuction.ID, endedAuction.SaleStatus)

		// Broadcast thông báo WebSocket cho các client đang xem trong phòng
		handlers.Hub.Broadcast(endedAuction.ID, handlers.WSMessage{
			Event: "auction_ended",
			Payload: map[string]interface{}{
				"auction_id":  endedAuction.ID,
				"message":     "Phiên đấu giá đã kết thúc!",
				"ended_at":    now.Format(time.RFC3339),
				"sale_status": endedAuction.SaleStatus,
			},
		})
	}
}
