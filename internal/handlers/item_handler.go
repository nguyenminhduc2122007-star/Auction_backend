package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"auction-backend/internal/services"
	"auction-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	itemService *services.ItemService
}

func NewItemHandler(itemService *services.ItemService) *ItemHandler {
	return &ItemHandler{itemService: itemService}
}

type CreateItemRequest struct {
	Title        string  `json:"title" binding:"required"`
	Type         string  `json:"type"`
	Category     string  `json:"category"` // Bổ sung để hứng trường category từ phía Frontend
	Status       string  `json:"status"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	StartPrice   float64 `json:"start_price"`
	CurrentPrice float64 `json:"current_price"`
}

type UpdateItemRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gte=0"`
}

func (h *ItemHandler) CreateItem(c *gin.Context) {
	var req CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// Fallback Type từ Category nếu Type trống
	itemType := req.Type
	if itemType == "" {
		itemType = req.Category
	}

	// Tự động gán StartPrice và CurrentPrice nếu người dùng chỉ truyền Price
	startPrice := req.StartPrice
	if startPrice == 0 && req.Price > 0 {
		startPrice = req.Price
	}
	currentPrice := req.CurrentPrice
	if currentPrice == 0 && req.Price > 0 {
		currentPrice = req.Price
	}

	item, err := h.itemService.CreateItem(services.CreateItemInput{
		Title:        req.Title,
		Type:         itemType,
		Status:       req.Status,
		Description:  req.Description,
		Price:        req.Price,
		StartPrice:   startPrice,
		CurrentPrice: currentPrice,
	})
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Success(c, http.StatusCreated, "Thành công", item)
}

// SellerCreateItem creates an auction using the seller identity from token
func (h *ItemHandler) SellerCreateItem(c *gin.Context) {
	var req CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	uid, ok := c.Get("user_id")
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthenticated")
		return
	}
	sellerID, ok := uid.(uint)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "invalid user id")
		return
	}

	itemType := req.Type
	if itemType == "" {
		itemType = req.Category
	}

	startPrice := req.StartPrice
	if startPrice == 0 && req.Price > 0 {
		startPrice = req.Price
	}
	currentPrice := req.CurrentPrice
	if currentPrice == 0 && req.Price > 0 {
		currentPrice = req.Price
	}

	item, err := h.itemService.CreateItemForSeller(sellerID, services.CreateItemInput{
		Title:        req.Title,
		Type:         itemType,
		Status:       req.Status,
		Description:  req.Description,
		Price:        req.Price,
		StartPrice:   startPrice,
		CurrentPrice: currentPrice,
	})
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Success(c, http.StatusCreated, "Thành công", item)
}

// SellerUpdateItem updates an auction owned by the seller if no bids placed
func (h *ItemHandler) SellerUpdateItem(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	uid, ok := c.Get("user_id")
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthenticated")
		return
	}
	sellerID, ok := uid.(uint)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "invalid user id")
		return
	}

	var req UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	item, err := h.itemService.UpdateItemBySeller(sellerID, id, req.Title, req.Description, req.Price)
	if err != nil {
		handleItemError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Thành công", item)
}

// SellerListItems returns auctions that belong to authenticated seller
func (h *ItemHandler) SellerListItems(c *gin.Context) {
	uid, ok := c.Get("user_id")
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthenticated")
		return
	}
	sellerID, ok := uid.(uint)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "invalid user id")
		return
	}
	items, err := h.itemService.GetAllItemsBySeller(sellerID)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "Thành công", items)
}

// AdminListItems supports q, status, pagination
func (h *ItemHandler) AdminListItems(c *gin.Context) {
	page, err := parsePositiveInt(c.DefaultQuery("page", "1"), 1)
	if err != nil {
		utils.BadRequest(c, "page must be a positive number")
		return
	}

	limit, err := parsePositiveInt(c.DefaultQuery("limit", "10"), 10)
	if err != nil {
		utils.BadRequest(c, "limit must be a positive number")
		return
	}

	status := c.Query("status")

	result, err := h.itemService.AdminListItems(services.ItemListQuery{
		Q:     c.Query("q"),
		Page:  page,
		Limit: limit,
	}, status)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "Thành công", result)
}

// AdminUpdateItemStatus updates status (active/rejected/suspended)
func (h *ItemHandler) AdminUpdateItemStatus(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	if body.Status == "" {
		utils.BadRequest(c, "status is required")
		return
	}

	if err := h.itemService.AdminUpdateItemStatus(id, body.Status); err != nil {
		handleItemError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Thành công", nil)
}

// AdminDeleteItem deletes an item
func (h *ItemHandler) AdminDeleteItem(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	if err := h.itemService.DeleteItem(id); err != nil {
		handleItemError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Thành công", nil)
}

func (h *ItemHandler) GetItem(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	item, err := h.itemService.GetItemByID(id)
	if err != nil {
		handleItemError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Thành công", item)
}

// CommonCreateItem dispatches create based on authenticated role
func (h *ItemHandler) CommonCreateItem(c *gin.Context) {
	r, ok := c.Get("role")
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthenticated")
		return
	}
	role, _ := r.(string)
	switch role {
	case "seller", "bidder":
		h.SellerCreateItem(c)
		return
	case "admin":
		h.CreateItem(c)
		return
	default:
		utils.Error(c, http.StatusForbidden, "forbidden")
		return
	}
}

// CommonUpdateItem dispatches update based on role
func (h *ItemHandler) CommonUpdateItem(c *gin.Context) {
	r, ok := c.Get("role")
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthenticated")
		return
	}
	role, _ := r.(string)
	switch role {
	case "seller", "bidder":
		h.SellerUpdateItem(c)
		return
	case "admin":
		h.UpdateItem(c)
		return
	default:
		utils.Error(c, http.StatusForbidden, "forbidden")
		return
	}
}

// CommonListItems dispatches list based on role
func (h *ItemHandler) CommonListItems(c *gin.Context) {
	r, ok := c.Get("role")
	if !ok {
		h.ListItems(c)
		return
	}
	role, _ := r.(string)
	switch role {
	case "seller":
		h.SellerListItems(c)
		return
	case "admin":
		h.AdminListItems(c)
		return
	default:
		h.ListItems(c)
		return
	}
}

// CommonDeleteItem dispatches delete based on role
func (h *ItemHandler) CommonDeleteItem(c *gin.Context) {
	r, ok := c.Get("role")
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthenticated")
		return
	}
	role, _ := r.(string)
	switch role {
	case "seller", "bidder":
		h.DeleteItem(c)
		return
	case "admin":
		h.AdminDeleteItem(c)
		return
	default:
		utils.Error(c, http.StatusForbidden, "forbidden")
		return
	}
}

func (h *ItemHandler) ListItems(c *gin.Context) {
	page, err := parsePositiveInt(c.DefaultQuery("page", "1"), 1)
	if err != nil {
		utils.BadRequest(c, "page must be a positive number")
		return
	}

	limit, err := parsePositiveInt(c.DefaultQuery("limit", "10"), 10)
	if err != nil {
		utils.BadRequest(c, "limit must be a positive number")
		return
	}

	result, err := h.itemService.ListItems(services.ItemListQuery{
		Q:     c.Query("q"),
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "Thành công", result)
}

func (h *ItemHandler) GetSellerItems(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		userIDStr = c.Param("user_id")
	}

	userID, err := utils.ParseUintID(userIDStr)
	if err != nil {
		utils.BadRequest(c, "invalid user_id")
		return
	}

	items, err := h.itemService.GetAllItemsBySeller(userID)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "Thành công", items)
}

func (h *ItemHandler) DashboardStats(c *gin.Context) {
	stats, err := h.itemService.GetDashboardStats()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "Thành công", stats)
}

func (h *ItemHandler) UpdateItem(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var req UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	item, err := h.itemService.UpdateItem(id, req.Title, req.Description, req.Price)
	if err != nil {
		handleItemError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Thành công", item)
}

func (h *ItemHandler) DeleteItem(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if err := h.itemService.DeleteItem(id); err != nil {
		handleItemError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Thành công", nil)
}

func handleItemError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrItemNotFound) {
		utils.Error(c, http.StatusNotFound, err.Error())
		return
	}
	utils.BadRequest(c, err.Error())
}

func parsePositiveInt(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, errors.New("invalid positive integer")
	}
	return value, nil
}