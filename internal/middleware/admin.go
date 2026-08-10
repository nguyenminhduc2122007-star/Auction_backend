package middleware

import (
	"net/http"
	"strings"

	"auction-backend/internal/models"
	"auction-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// RequireAdmin blocks requests where the JWT role is not admin
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Thử lấy từ context key "role" hoặc "user_type"
		r, ok := c.Get("role")
		if !ok {
			r, ok = c.Get("user_type")
		}

		if !ok {
			utils.Error(c, http.StatusUnauthorized, "unauthenticated: role not found in context")
			c.Abort()
			return
		}

		roleStr, ok := r.(string)
		if !ok || roleStr == "" {
			utils.Error(c, http.StatusUnauthorized, "unauthenticated: invalid role format")
			c.Abort()
			return
		}

		// 2. So sánh không phân biệt hoa thường với Constant models.UserTypeAdmin
		isAdmin := strings.EqualFold(roleStr, string(models.UserTypeAdmin)) || 
			strings.EqualFold(roleStr, "admin")

		if !isAdmin {
			utils.Error(c, http.StatusForbidden, "forbidden: admin privileges required")
			c.Abort()
			return
		}

		c.Next()
	}
}