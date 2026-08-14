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

	// Khởi chạy Background Worker[cite: 17]
	auctionWorker := worker.NewAuctionWorker(db)
	auctionWorker.Start(10 * time.Second)

	// Repositories[cite: 17]
	userRepo := repository.NewUserRepository(db)
	auctionRepo := repository.NewAuctionRepository(db)

	// Services[cite: 17]
	authService := services.NewAuthService(userRepo)
	userService := services.NewUserService(userRepo)
	auctionService := services.NewAuctionService(auctionRepo)

	// Handlers[cite: 17]
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

	// 1. Authentication Group (Public)[cite: 17]
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", authHandler.Logout)
	}

	// 2. Auctions Group (Toàn bộ luồng Đấu giá & Sản phẩm)[cite: 17]
	auctions := api.Group("/auctions")
	{
		// --- Public / Reading API ---[cite: 17]
		auctions.GET("", auctionHandler.ListAuctions)

		// --- Seller / Draft Workflow ---[cite: 17]
		auctions.GET("/seller-eligibility", middleware.AuthMiddleware(), auctionHandler.CheckEligibility)
		auctions.POST("/drafts", middleware.AuthMiddleware(), auctionHandler.CreateDraft)
		auctions.PUT("/drafts/:id", middleware.AuthMiddleware(), auctionHandler.UpdateDraft)
		auctions.PUT("/drafts/:id/pricing", middleware.AuthMiddleware(), auctionHandler.UpdatePricing)
		auctions.GET("/drafts/:id/preview", middleware.AuthMiddleware(), auctionHandler.GetDraftPreview)
		auctions.POST("/drafts/:id/publish", middleware.AuthMiddleware(), auctionHandler.Publish)

		// --- Dynamic ID API ---[cite: 17]
		auctions.GET("/:id", auctionHandler.GetAuctionDetail)
		auctions.PATCH("/:id/status", middleware.AuthMiddleware(), auctionHandler.UpdateStatus)
		auctions.PATCH("/:id/approve", middleware.AuthMiddleware(), auctionHandler.ApproveAuction) // Route phê duyệt[cite: 17]
		auctions.DELETE("/:id", middleware.AuthMiddleware(), auctionHandler.DeleteAuction)

		// Realtime WebSocket[cite: 17]
		auctions.GET("/:id/ws", wsHandler.ServeWS)
	}

	// 3. Admin Group[cite: 17]
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.RequireAdmin())
	{
		admin.GET("/stats", auctionHandler.DashboardStats)
		admin.GET("/dashboard/stats", auctionHandler.DashboardStats)
		admin.GET("/users", userHandler.ListUsers)
		admin.PUT("/users/:id/role", userHandler.UpdateUserRole)
		admin.GET("/auctions", auctionHandler.AdminListAuctions) // Trả về toàn bộ danh sách cho trang quản trị[cite: 17]
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