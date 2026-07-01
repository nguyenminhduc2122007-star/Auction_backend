package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"auction-backend/internal/database"
	"auction-backend/internal/handlers"
	"auction-backend/internal/middleware"
	"auction-backend/internal/repository"
	"auction-backend/internal/services"
)

func main() {
	loadEnv()

	if err := database.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	if err := database.MigrateModels(); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	db := database.GetDB()
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authService)

	itemRepo := repository.NewItemRepository(db)
	itemService := services.NewItemService(itemRepo)
	itemHandler := handlers.NewItemHandler(itemService)

	router := gin.Default()
	auth := router.Group("/api/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	items := router.Group("/api/items")
	items.Use(middleware.AuthMiddleware())
	{
		items.POST("", itemHandler.CreateItem)
		items.GET("", itemHandler.ListItems)
		items.GET("/:id", itemHandler.GetItem)
		items.PUT("/:id", itemHandler.UpdateItem)
		items.DELETE("/:id", itemHandler.DeleteItem)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// loadEnv tìm thư mục gốc project (chứa go.mod) rồi load .env từ đó.
// Hoạt động đúng dù chạy từ bất kỳ thư mục nào.
func loadEnv() {
	root, err := findProjectRoot()
	if err != nil {
		log.Println("Could not find project root, using environment variables")
		return
	}
	envPath := filepath.Join(root, ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Println("No .env file found, using environment variables")
		return
	}
	log.Printf("Loaded env from: %s", envPath)
}

// findProjectRoot leo từ thư mục hiện tại lên đến khi tìm thấy go.mod
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
			break // đã lên đến filesystem root
		}
		dir = parent
	}
	return "", os.ErrNotExist
}