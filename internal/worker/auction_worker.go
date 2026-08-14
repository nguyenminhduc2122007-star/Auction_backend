package worker

import (
	"log"
	"time"

	"auction-backend/internal/handlers"
	"auction-backend/internal/models"

	"gorm.io/gorm"
)

type AuctionWorker struct {
	db       *gorm.DB
	stopChan chan struct{}
}

func NewAuctionWorker(db *gorm.DB) *AuctionWorker {
	return &AuctionWorker{
		db:       db,
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
		// Cập nhật trạng thái thành ENDED an toàn qua Gorm Model
		err := w.db.Model(&models.Auction{}).Where("id = ?", auction.ID).Update("status", models.AuctionStatusEnded).Error
		if err != nil {
			log.Printf("[Worker Error] Failed to end auction ID %d: %v", auction.ID, err)
			continue
		}

		log.Printf("[Worker] Auction ID %d ended successfully", auction.ID)

		// Broadcast thông báo WebSocket cho các client đang xem trong phòng
		handlers.Hub.Broadcast(auction.ID, handlers.WSMessage{
			Event: "auction_ended",
			Payload: map[string]interface{}{
				"auction_id": auction.ID,
				"message":    "Phiên đấu giá đã kết thúc!",
				"ended_at":   now.Format(time.RFC3339),
			},
		})
	}
}