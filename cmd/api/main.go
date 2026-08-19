package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"auction-backend/internal/database"
	"auction-backend/internal/handlers"
	"auction-backend/internal/middleware"
	"auction-backend/internal/repository"
	"auction-backend/internal/services"
	"auction-backend/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	loadEnv()

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatalf("missing required environment variable: JWT_SECRET")
	}

	if err := database.InitDB(); err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	if err := database.MigrateModels(); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	db := database.GetDB()

	// Khởi chạy Background Worker
	auctionWorker := worker.NewAuctionWorker(db)
	auctionWorker.Start(10 * time.Second)

	// Repositories
	userRepo := repository.NewUserRepository(db)
	auctionRepo := repository.NewAuctionRepository(db)

	// Services
	authService := services.NewAuthService(userRepo)
	userService := services.NewUserService(userRepo)
	auctionService := services.NewAuctionService(auctionRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	auctionHandler := handlers.NewAuctionHandler(auctionService)
	wsHandler := handlers.NewWSHandler(auctionService)

	router := gin.Default()
	router.Use(corsMiddleware())

	registerRoutes(router, userHandler, authHandler, auctionHandler, wsHandler)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Server running on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func registerRoutes(
	router *gin.Engine,
	userHandler *handlers.UserHandler,
	authHandler *handlers.AuthHandler,
	auctionHandler *handlers.AuctionHandler,
	wsHandler *handlers.WSHandler,
) {
	api := router.Group("/api")

	// 1. Authentication Group
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", authHandler.Logout)
	}

	// 2. User Profile Group
	users := api.Group("/users")
	users.Use(middleware.AuthMiddleware())
	{
		users.GET("/me/profile", userHandler.GetProfile)
		users.PUT("/me/profile", userHandler.UpdateProfile)
		users.GET("/me/bids", userHandler.GetMyBids)
		users.GET("/me/won-auctions", userHandler.GetWonAuctions)
		users.GET("/me/my-auctions", auctionHandler.ListMyAuctions)
	}

	// 3. Auctions & Bidding Group
	auctions := api.Group("/auctions")
	{
		// Public Endpoints
		auctions.GET("", auctionHandler.ListAuctions)
		auctions.GET("/:id", auctionHandler.GetAuctionDetail)
		auctions.GET("/:id/ws", wsHandler.ServeWS) // WebSocket Realtime Channel

		// Authenticated Seller & Buyer Endpoints
		auctions.GET("/mine", middleware.AuthMiddleware(), auctionHandler.ListMyAuctions)
		auctions.GET("/seller-eligibility", middleware.AuthMiddleware(), auctionHandler.CheckEligibility)
		auctions.POST("/:id/bids", middleware.AuthMiddleware(), auctionHandler.PlaceBid)

		// Draft Management
		auctions.POST("/drafts", middleware.AuthMiddleware(), auctionHandler.CreateDraft)
		auctions.PUT("/drafts/:id", middleware.AuthMiddleware(), auctionHandler.UpdateDraft)
		auctions.PUT("/drafts/:id/pricing", middleware.AuthMiddleware(), auctionHandler.UpdatePricing)
		auctions.GET("/drafts/:id/preview", middleware.AuthMiddleware(), auctionHandler.GetDraftPreview)
		auctions.POST("/drafts/:id/publish", middleware.AuthMiddleware(), auctionHandler.Publish)

		// Auction Actions
		auctions.PATCH("/:id/status", middleware.AuthMiddleware(), auctionHandler.UpdateStatus)
		auctions.DELETE("/:id", middleware.AuthMiddleware(), auctionHandler.DeleteAuction)
		auctions.POST("/:id/relist", middleware.AuthMiddleware(), auctionHandler.RelistAuction)
	}

	// 4. Admin Management Group (Nâng cấp Bước 4 & Bước 5)
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.RequireAdmin())
	{
		// Dashboard & Stats
		admin.GET("/stats", auctionHandler.DashboardStats)
		admin.GET("/dashboard/stats", auctionHandler.DashboardStats)

		// User Management
		admin.GET("/users", userHandler.ListUsers)
		admin.GET("/users/:id/summary", userHandler.GetUserSummary)
		admin.PUT("/users/:id/role", userHandler.UpdateUserRole)

		// Auction Management & Approval
		admin.GET("/auctions", auctionHandler.AdminListAuctions)
		admin.PATCH("/auctions/:id/approve", auctionHandler.ApproveAuction)
		admin.PATCH("/auctions/:id/reject", auctionHandler.RejectAuction)

		// 🟢 BƯỚC 4: Các Route Admin điều khiển phiên đấu giá
		admin.POST("/auctions/:id/pause", auctionHandler.PauseAuction)
		admin.POST("/auctions/:id/resume", auctionHandler.ResumeAuction)
		admin.POST("/auctions/:id/cancel", auctionHandler.CancelAuction)

		// 🟢 Trigger đóng phiên hết hạn (dùng test trên Postman)
		admin.POST("/auctions/close-expired", auctionHandler.TriggerCloseExpired)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func loadEnv() {
	dir, err := findProjectRoot()
	if err != nil {
		return
	}
	_ = godotenv.Load(filepath.Join(dir, ".env"))
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}
