package middleware

import (
	"net/http"
	"strings"

	"auction-backend/internal/models"
	"auction-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// RequireAdmin kiểm tra xem người dùng thực hiện request có quyền Admin hay không
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Thử lấy giá trị vai trò từ context key "role" hoặc "user_type"
		r, ok := c.Get("role")
		if !ok {
			r, ok = c.Get("user_type")
		}

		if !ok {
			utils.Error(c, http.StatusUnauthorized, "Thao tác thất bại: Không tìm thấy thông tin phân quyền")
			c.Abort()
			return
		}

		roleStr, ok := r.(string)
		if !ok || strings.TrimSpace(roleStr) == "" {
			utils.Error(c, http.StatusUnauthorized, "Định dạng thông tin phân quyền không hợp lệ")
			c.Abort()
			return
		}

		// 2. So sánh không phân biệt hoa thường với Constant UserTypeAdmin hoặc chuỗi "admin"
		isAdmin := strings.EqualFold(roleStr, string(models.UserTypeAdmin)) ||
			strings.EqualFold(roleStr, "admin")

		if !isAdmin {
			utils.Error(c, http.StatusForbidden, "Truy cập bị từ chối: Yêu cầu quyền Quản trị viên (Admin)")
			c.Abort()
			return
		}

		c.Next()
	}
}
