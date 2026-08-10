package middleware

import (
	"net/http"

	"auction-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// RequireAdmin blocks requests where the JWT role is not admin
func RequireAdmin() gin.HandlerFunc {
    return func(c *gin.Context) {
        r, ok := c.Get("role")
        if !ok {
            utils.Error(c, http.StatusUnauthorized, "unauthenticated")
            c.Abort()
            return
        }
        role, _ := r.(string)
        if role != "admin" {
            utils.Error(c, http.StatusForbidden, "forbidden: admin only")
            c.Abort()
            return
        }
        c.Next()
    }
}
