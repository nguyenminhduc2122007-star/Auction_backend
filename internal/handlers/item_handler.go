package handlers

import (
	"errors"
	"net/http"

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
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"gte=0"`
}


type UpdateItemRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"gte=0"`
}

type ListItemRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type GetItemRequest struct {
	ID uint `json:"id" binding:"required"`
}

type DeleteItemRequest struct {
	ID uint `json:"id" binding:"required"`
}

func (h *ItemHandler) CreateItem(c *gin.Context) {
	var req CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	item, err := h.itemService.CreateItem(userID, req.Title, req.Description, req.Price)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Item created successfully", "data": item})
}

func (h *ItemHandler) GetItem(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.itemService.GetItemByID(id)
	if err != nil {
		handleItemError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *ItemHandler) ListItems(c *gin.Context) {
	items, err := h.itemService.GetAllItems()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *ItemHandler) UpdateItem(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	item, err := h.itemService.UpdateItem(userID, id, req.Title, req.Description, req.Price)
	if err != nil {
		handleItemError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item updated successfully", "data": item})
}

func (h *ItemHandler) DeleteItem(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	if err := h.itemService.DeleteItem(userID, id); err != nil {
		handleItemError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item deleted successfully"})
}

// getUserID lấy user_id (uint) đã được AuthMiddleware set vào context.
func getUserID(c *gin.Context) (uint, bool) {
	uid, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	userID, ok := uid.(uint)
	return userID, ok
}

// handleItemError map lỗi từ service sang HTTP status code phù hợp.
func handleItemError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrItemNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrPermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}