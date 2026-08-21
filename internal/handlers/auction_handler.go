package handlers

import (
	"net/http"
	"strconv"
	"time"

	"auction-backend/internal/models"
	"auction-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type AuctionHandler struct {
	service *services.AuctionService
}

func NewAuctionHandler(service *services.AuctionService) *AuctionHandler {
	return &AuctionHandler{service: service}
}

type UpdateStatusInput struct {
	Status string `json:"status" binding:"required"`
}

type RejectAuctionInput struct {
	Reason string `json:"reason"`
}

type CancelAuctionInput struct {
	Reason string `json:"reason" binding:"required"`
}

type RelistAuctionInput struct {
	StartAt time.Time `json:"start_at" binding:"required"`
	EndAt   time.Time `json:"end_at" binding:"required"`
}

type PlaceBidInput struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

func getUserIDFromContext(c *gin.Context) (uint, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		val, exists = c.Get("userID")
	}

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return 0, false
	}

	switch v := val.(type) {
	case uint:
		return v, true
	case uint64:
		return uint(v), true
	case float64:
		return uint(v), true
	case int:
		return uint(v), true
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
		return 0, false
	}
}

func (h *AuctionHandler) writeAuctionError(c *gin.Context, err error) {
	if err == services.ErrAuctionNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err == services.ErrUnauthorized {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

// POST /api/auctions/:id/bids
func (h *AuctionHandler) PlaceBid(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	auctionID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID phiên đấu giá không hợp lệ"})
		return
	}

	var input PlaceBidInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Số tiền đặt giá không hợp lệ"})
		return
	}

	bid, extended, newEndAt, err := h.service.ProcessBid(uint(auctionID), userID, input.Amount)
	if err != nil {
		h.writeAuctionError(c, err)
		return
	}

	payload := gin.H{
		"auction_id":    auctionID,
		"bid_id":        bid.ID,
		"bidder_id":     bid.BidderID,
		"current_price": bid.Amount,
		"created_at":    bid.CreatedAt,
		"is_extended":   extended,
	}
	if extended && newEndAt != nil {
		payload["new_end_at"] = newEndAt
	}

	Hub.Broadcast(uint(auctionID), WSMessage{
		Event:   "bid:placed",
		Payload: payload,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Đặt giá thành công",
		"data": gin.H{
			"bid":         bid,
			"is_extended": extended,
			"new_end_at":  newEndAt,
		},
	})
}

// GET /api/auctions
func (h *AuctionHandler) ListAuctions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")
	catIDStr := c.Query("category_id")

	var categoryID uint
	if catIDStr != "" {
		if id, err := strconv.ParseUint(catIDStr, 10, 32); err == nil {
			categoryID = uint(id)
		}
	}

	auctions, total, err := h.service.ListAuctions(page, limit, status, categoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    auctions,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// GET /api/auctions/:id
func (h *AuctionHandler) GetAuctionDetail(c *gin.Context) {
	idParam := c.Param("id")
	auctionID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auction ID"})
		return
	}

	auction, err := h.service.GetAuctionDetail(uint(auctionID))
	if err != nil {
		h.writeAuctionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    auction,
	})
}

// PATCH /api/auctions/:id/status
func (h *AuctionHandler) UpdateStatus(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	auctionID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auction ID"})
		return
	}

	var input UpdateStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu trạng thái không hợp lệ"})
		return
	}

	err = h.service.UpdateStatus(userID, uint(auctionID), input.Status)
	if err != nil {
		h.writeAuctionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cập nhật trạng thái thành công",
		"status":  input.Status,
	})
}

// POST /api/admin/auctions/:id/approve
func (h *AuctionHandler) ApproveAuction(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	auctionID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auction ID"})
		return
	}

	err = h.service.ApproveAuction(userID, uint(auctionID))
	if err != nil {
		h.writeAuctionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Phê duyệt phiên đấu giá thành công",
	})
}

// POST /api/admin/auctions/:id/reject
func (h *AuctionHandler) RejectAuction(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}
	auctionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auction ID"})
		return
	}
	var input RejectAuctionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rejection payload"})
		return
	}
	if err := h.service.RejectAuction(userID, uint(auctionID), input.Reason); err != nil {
		h.writeAuctionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Auction rejected"})
}

// POST /api/admin/auctions/:id/pause
func (h *AuctionHandler) PauseAuction(c *gin.Context) {
	adminID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	auctionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID phiên đấu giá không hợp lệ"})
		return
	}

	if err := h.service.PauseAuction(adminID, uint(auctionID)); err != nil {
		h.writeAuctionError(c, err)
		return
	}

	Hub.Broadcast(uint(auctionID), WSMessage{
		Event: "auction:status_changed",
		Payload: gin.H{
			"auction_id": auctionID,
			"status":     models.AuctionStatusPaused,
			"message":    "Phiên đấu giá vừa bị TẠM DỪNG bởi Quản trị viên",
		},
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã tạm dừng phiên đấu giá thành công"})
}

// POST /api/admin/auctions/:id/resume
func (h *AuctionHandler) ResumeAuction(c *gin.Context) {
	adminID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	auctionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID phiên đấu giá không hợp lệ"})
		return
	}

	if err := h.service.ResumeAuction(adminID, uint(auctionID)); err != nil {
		h.writeAuctionError(c, err)
		return
	}

	Hub.Broadcast(uint(auctionID), WSMessage{
		Event: "auction:status_changed",
		Payload: gin.H{
			"auction_id": auctionID,
			"status":     models.AuctionStatusLive,
			"message":    "Phiên đấu giá đã được MỞ LẠI",
		},
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã tiếp tục phiên đấu giá thành công"})
}

// POST /api/admin/auctions/:id/cancel
func (h *AuctionHandler) CancelAuction(c *gin.Context) {
	adminID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	auctionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID phiên đấu giá không hợp lệ"})
		return
	}

	var input CancelAuctionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng nhập lý do hủy phiên"})
		return
	}

	if err := h.service.CancelAuction(adminID, uint(auctionID), input.Reason); err != nil {
		h.writeAuctionError(c, err)
		return
	}

	Hub.Broadcast(uint(auctionID), WSMessage{
		Event: "auction:status_changed",
		Payload: gin.H{
			"auction_id": auctionID,
			"status":     models.AuctionStatusCancelled,
			"reason":     input.Reason,
			"message":    "Phiên đấu giá đã bị HỦY bởi Quản trị viên",
		},
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã hủy phiên đấu giá thành công"})
}

// POST /api/admin/auctions/close-expired
func (h *AuctionHandler) TriggerCloseExpired(c *gin.Context) {
	if err := h.service.CloseExpiredAuctions(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Đã quét và đóng các phiên đấu giá hết hạn thành công",
	})
}

// DELETE /api/auctions/:id
func (h *AuctionHandler) DeleteAuction(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	auctionID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auction ID"})
		return
	}

	err = h.service.DeleteAuction(userID, uint(auctionID))
	if err != nil {
		h.writeAuctionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Xóa phiên đấu giá thành công",
	})
}

// GET /api/admin/auctions
func (h *AuctionHandler) AdminListAuctions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	auctions, total, err := h.service.AdminListAuctions(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    auctions,
		"total":   total,
	})
}

// GET /api/admin/dashboard/stats
func (h *AuctionHandler) DashboardStats(c *gin.Context) {
	stats, err := h.service.GetDashboardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GET /api/auctions/seller-eligibility
func (h *AuctionHandler) CheckEligibility(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	res, err := h.service.CheckSellerEligibility(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// POST /api/auctions/drafts
func (h *AuctionHandler) CreateDraft(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var input services.SaveDraftInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	auction, err := h.service.CreateDraft(userID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": auction})
}

// PUT /api/auctions/drafts/:id
func (h *AuctionHandler) UpdateDraft(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	auctionID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auction ID"})
		return
	}

	var input services.SaveDraftInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	auction, err := h.service.UpdateDraft(userID, uint(auctionID), input)
	if err != nil {
		h.writeAuctionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": auction})
}

// PUT /api/auctions/drafts/:id/pricing
func (h *AuctionHandler) UpdatePricing(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	auctionID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auction ID"})
		return
	}

	var input services.PricingConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pricing, err := h.service.UpdatePricing(userID, uint(auctionID), input)
	if err != nil {
		h.writeAuctionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": pricing})
}

// GET /api/auctions/drafts/:id/preview
func (h *AuctionHandler) GetDraftPreview(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	auctionID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auction ID"})
		return
	}

	auction, err := h.service.GetDraftPreview(userID, uint(auctionID))
	if err != nil {
		h.writeAuctionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": auction})
}

// POST /api/auctions/drafts/:id/publish
func (h *AuctionHandler) Publish(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	auctionID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auction ID"})
		return
	}

	var input models.PublishAuctionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	auction, err := h.service.PublishAuction(userID, uint(auctionID), input.StartAt, input.EndAt)
	if err != nil {
		h.writeAuctionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Phiên đấu giá đã xuất bản thành công!",
		"data":    auction,
	})
}

// POST /api/auctions/:id/relist
func (h *AuctionHandler) RelistAuction(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}
	auctionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auction ID"})
		return
	}
	var input RelistAuctionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relist payload"})
		return
	}
	auction, err := h.service.RelistAuction(userID, uint(auctionID), input.StartAt, input.EndAt)
	if err != nil {
		h.writeAuctionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": auction})
}

// GET /api/seller/auctions
func (h *AuctionHandler) ListMyAuctions(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.DefaultQuery("status", "ALL")
	auctions, total, err := h.service.ListSellerAuctions(userID, status, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": auctions, "total": total, "page": page, "limit": limit})
}
