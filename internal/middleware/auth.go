package middleware

import (
	"net/http"
	"strings"

	"auction-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware kiểm tra JWT Token từ Cookie hoặc Header Authorization.
// Bắt buộc request phải có Token hợp lệ, nếu không sẽ chặn lại và trả về 401.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ""

		// 1. Ưu tiên kiểm tra Token trong Cookie "token"
		if cookieToken, err := c.Cookie("token"); err == nil && cookieToken != "" {
			tokenString = cookieToken
		} else {
			// 2. Nếu không thấy Cookie, fallback đọc từ Header Authorization
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// ❌ 3. Nếu không tìm thấy Token ở cả Cookie lẫn Header -> Chặn ngay lập tức
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "unauthenticated",
			})
			c.Abort()
			return
		}

		// ❌ 4. Giải mã và validate Token -> Nếu Token hỏng/hết hạn thì chặn ngay
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "unauthenticated",
			})
			c.Abort()
			return
		}

		// ✅ 5. Token chuẩn -> Lưu thông tin User vào Context để các Handler phía sau sử dụng
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.UserType)      // e.g. "admin" hoặc "seller"
		c.Set("user_type", claims.UserType) // giữ tương thích ngược

		c.Next()
	}
}