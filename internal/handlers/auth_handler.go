package handlers

import (
	"net/http"
	"os"

	"auction-backend/internal/models"
	"auction-backend/internal/services"
	"auction-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	FullName string `json:"full_name"`
	Name     string `json:"name"`
	UserType string `json:"user_type"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// support both `full_name` and legacy `name`
	fullName := req.FullName
	if fullName == "" {
		fullName = req.Name
	}
	// support both `user_type` and legacy `role`
	userType := req.UserType
	if userType == "" {
		userType = req.Role
	}
	if fullName == "" || userType == "" {
		utils.BadRequest(c, "full_name (or name) and user_type (or role) are required")
		return
	}

	user, err := h.authService.Register(req.Email, req.Password, fullName, models.UserType(userType))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	token, err := utils.GenerateToken(user.ID, string(user.UserType))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to generate token")
		return
	}

	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	if cookieDomain == "" {
		cookieDomain = "localhost"
	}
	c.SetCookie("token", token, 86400, "/", cookieDomain, false, true)

	utils.Success(c, http.StatusCreated, "Thành công", gin.H{
		"token": token,
		"user":  user,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	token, user, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	if cookieDomain == "" {
		cookieDomain = "localhost"
	}
	c.SetCookie("token", token, 86400, "/", cookieDomain, false, true)

	utils.Success(c, http.StatusOK, "Thành công", gin.H{
		"token": token,
		"user":  user,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	if cookieDomain == "" {
		cookieDomain = "localhost"
	}
	c.SetCookie("token", "", -1, "/", cookieDomain, false, true)
	utils.Success(c, http.StatusOK, "Đăng xuất thành công", nil)
}
