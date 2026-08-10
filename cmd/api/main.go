package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"auction-backend/internal/database"
	"auction-backend/internal/handlers"
	"auction-backend/internal/middleware"
	"auction-backend/internal/repository"
	"auction-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	loadEnv()

	// enforce presence of critical secrets/config to avoid insecure defaults
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

	userRepo := repository.NewUserRepository(db)
	authHandler := handlers.NewAuthHandler(services.NewAuthService(userRepo))
	userHandler := handlers.NewUserHandler(services.NewUserService(userRepo))

	itemRepo := repository.NewItemRepository(db)
	itemHandler := handlers.NewItemHandler(services.NewItemService(itemRepo, userRepo))

	router := gin.Default()
	router.Use(corsMiddleware())

	// register consolidated routes: /api/auth, /api/items and /api/admin
	registerRoutes(router, itemHandler, userHandler, authHandler)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Server running on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// registerRoutes sets up the consolidated RESTful API routes per spec
func registerRoutes(router *gin.Engine, itemHandler *handlers.ItemHandler, userHandler *handlers.UserHandler, authHandler *handlers.AuthHandler) {
	api := router.Group("/api")

	// auth group (public)
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", authHandler.Logout)
	}

	// items group
	items := api.Group("/items")
	{
		// public list and get details
		items.GET("", itemHandler.CommonListItems)
		items.GET("/:id", itemHandler.GetItem)

		// authenticated item operations
		items.POST("", middleware.AuthMiddleware(), itemHandler.CommonCreateItem)
		items.PUT("/:id", middleware.AuthMiddleware(), itemHandler.CommonUpdateItem)
		items.DELETE("/:id", middleware.AuthMiddleware(), itemHandler.CommonDeleteItem)
		items.PUT("/:id/status", middleware.AuthMiddleware(), middleware.RequireAdmin(), itemHandler.AdminUpdateItemStatus)
	}

	// admin-only routes (require admin)
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.RequireAdmin())
	{
		admin.GET("/stats", itemHandler.DashboardStats)
		admin.GET("/dashboard/stats", itemHandler.DashboardStats)
		admin.GET("/users", userHandler.ListUsers)
		admin.PUT("/users/:id/role", userHandler.UpdateUserRole)
		admin.GET("/auctions", itemHandler.AdminListItems)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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
