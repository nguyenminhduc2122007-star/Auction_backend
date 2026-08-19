package middleware

import (
	"net/http"
	"strconv"

	"auction-backend/internal/models"
	"auction-backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequireAuctionLive kiểm tra xem phiên đấu giá có đang ở trạng thái LIVE hay không
func RequireAuctionLive(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Lấy auction_id từ URL Param (:id hoặc :auction_id)
		auctionIDStr := c.Param("id")
		if auctionIDStr == "" {
			auctionIDStr = c.Param("auction_id")
		}

		var auctionID uint64
		var err error

		if auctionIDStr != "" {
			auctionID, err = strconv.ParseUint(auctionIDStr, 10, 64)
		}

		if err != nil || auctionID == 0 {
			utils.Error(c, http.StatusBadRequest, "ID phiên đấu giá không hợp lệ")
			c.Abort()
			return
		}

		// 2. Query nhanh kiểm tra thông tin và trạng thái phiên
		var auction models.Auction
		if err := db.Select("id", "status", "end_at", "current_price").First(&auction, auctionID).Error; err != nil {
			utils.Error(c, http.StatusNotFound, "Không tìm thấy phiên đấu giá")
			c.Abort()
			return
		}

		// 3. Kiểm tra nếu phiên bị TẠM DỪNG (PAUSED)
		if auction.Status == models.AuctionStatusPaused {
			utils.Error(c, http.StatusBadRequest, "Phiên đấu giá đang bị TẠM DỪNG bởi Quản trị viên, không thể đặt giá")
			c.Abort()
			return
		}

		// 4. Kiểm tra nếu phiên chưa bắt đầu hoặc đã kết thúc
		if auction.Status != models.AuctionStatusLive {
			utils.Error(c, http.StatusBadRequest, "Phiên đấu giá hiện không trong thời gian diễn ra")
			c.Abort()
			return
		}

		// 5. Lưu object auction vào Context để Handler phía sau dùng trực tiếp (không cần Query DB lại)
		c.Set("current_auction", &auction)
		c.Next()
	}
}
