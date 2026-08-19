package middleware

import (
	"net/http"
	"sync"
	"time"

	"auction-backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Quản lý bộ giới hạn tần suất theo User ID
type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	clients   = make(map[uint]*clientLimiter)
	clientsMu sync.Mutex
)

// Tự động dọn dẹp các limiter không active sau 3 phút
func init() {
	go func() {
		for {
			time.Sleep(3 * time.Minute)
			clientsMu.Lock()
			for userID, cl := range clients {
				if time.Since(cl.lastSeen) > 3*time.Minute {
					delete(clients, userID)
				}
			}
			clientsMu.Unlock()
		}
	}()
}

// BidRateLimiter giới hạn 2 request / giây, burst tối đa 5
func BidRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		userID, ok := userIDVal.(uint)
		if !ok {
			c.Next()
			return
		}

		clientsMu.Lock()
		cl, found := clients[userID]
		if !found {
			// Cho phép 2 token / giây, burst = 5
			limiter := rate.NewLimiter(rate.Limit(2), 5)
			cl = &clientLimiter{limiter: limiter, lastSeen: time.Now()}
			clients[userID] = cl
		} else {
			cl.lastSeen = time.Now()
		}
		clientsMu.Unlock()

		if !cl.limiter.Allow() {
			utils.Error(c, http.StatusTooManyRequests, "Bạn thao tác quá nhanh. Vui lòng thử lại sau vài giây.")
			c.Abort()
			return
		}

		c.Next()
	}
}
