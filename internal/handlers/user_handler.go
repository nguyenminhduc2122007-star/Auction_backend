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
	registered24h := c.Query("registered_24h") == "true"
	isSuspicious := c.Query("is_suspicious") == "true"
	search := c.Query("search")

	input := services.UserListFilterInput{
		Registered24h: registered24h,
		IsSuspicious:  isSuspicious,
		Search:        search,
	}

	users, err := h.userService.ListUsersFiltered(input)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "Lấy danh sách người dùng thành công", users)
}

func (h *UserHandler) GetUserSummary(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "ID người dùng không hợp lệ")
		return
	}

	summary, err := h.userService.GetUserSummary(id)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "Không tìm thấy thông tin tóm tắt người dùng")
		return
	}

	utils.Success(c, http.StatusOK, "Lấy tóm tắt người dùng thành công", summary)
}

type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	id, err := utils.ParseUintID(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "ID người dùng không hợp lệ")
		return
	}

	var req UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Trả về lỗi chi tiết thay vì thông báo chung chung
		utils.BadRequest(c, "Dữ liệu yêu cầu không hợp lệ: "+err.Error())
		return
	}

	user, err := h.userService.UpdateUserRole(id, req.Role)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "Cập nhật vai trò thành công", user)
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông tin cá nhân"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var req services.UpdateProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		// In ra chi tiết err.Error() để biết chính xác field nào bị lỗi JSON
		utils.BadRequest(c, "Dữ liệu cập nhật không hợp lệ: "+err.Error())
		return
	}

	user, err := h.userService.UpdateProfile(userID, req)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "Cập nhật thông tin cá nhân thành công", user)
}

func (h *UserHandler) GetMyBids(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	bids, err := h.userService.GetMyBids(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bids)
}

func (h *UserHandler) GetWonAuctions(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	won, err := h.userService.GetWonAuctions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, won)
}
