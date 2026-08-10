package middleware

import (
	"log"
	"net/http"
	"strings"

	"auction-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware kiểm tra JWT Token từ Header Authorization hoặc Cookie.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ""

		// 1. Ưu tiên đọc từ Header Authorization (Chuẩn cho Single Page Application / Vue)
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// 2. Nếu Header không có mới fallback tìm trong Cookie
		if tokenString == "" {
			if cookieToken, err := c.Cookie("token"); err == nil && cookieToken != "" {
				tokenString = cookieToken
			}
		}

		// 3. Nếu không tìm thấy Token ở cả 2 nơi -> Chặn
		if tokenString == "" {
			log.Println("[AuthMiddleware] ❌ Không tìm thấy Token trong Header hoặc Cookie")
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "unauthenticated",
			})
			c.Abort()
			return
		}

		// 4. Giải mã và Validate Token
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			// In log chính xác nguyên nhân lỗi ra Terminal Backend
			log.Printf("[AuthMiddleware] ❌ Validate Token thất bại: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "unauthenticated",
			})
			c.Abort()
			return
		}

		// 5. Token hợp lệ -> Gán dữ liệu vào Context
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.UserType)
		c.Set("user_type", claims.UserType)

		c.Next()
	}
}