package handlers

import (
	"net/http"

	"auction-backend/internal/services"
	"auction-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.userService.ListUsers()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "Thành công", users)
}

type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var req UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	user, err := h.userService.UpdateUserRole(id, req.Role)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "Thành công", user)
}
