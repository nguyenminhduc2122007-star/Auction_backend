package handlers

import (
	"net/http"

	"auction-backend/internal/services"

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

func (h *ItemHandler) CreateItem(c *gin.Context) {
	var req CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID, ok := uid.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	item, err := h.itemService.CreateItem(userID, req.Title, req.Description, req.Price)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Item created successfully",
		"data":    item,
	})
}