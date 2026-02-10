package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"grafikarsa/internal/config"
	"grafikarsa/internal/db"
	"grafikarsa/internal/handler"
	"grafikarsa/internal/middleware"
	"grafikarsa/internal/repository"
	"grafikarsa/internal/service"
)

func main() {
	// Load configuration
	cfg := config.Load()

	log.Printf("Starting Grafikarsa API (%s)...", cfg.AppEnv)

	// Connect to database
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Pool.Close()

	log.Println("Connected to database")

	// ==================== REPOSITORIES ====================
	authRepo := repository.NewAuthRepository(database.Pool)
	userRepo := repository.NewUserRepository(database.Pool)
	followRepo := repository.NewFollowRepository(database.Pool)
	portfolioRepo := repository.NewPortfolioRepository(database.Pool)
	tagRepo := repository.NewTagRepository(database.Pool)
	majorRepo := repository.NewMajorRepository(database.Pool)
	yearRepo := repository.NewAcademicYearRepository(database.Pool)
	classRepo := repository.NewClassRepository(database.Pool)
	uploadRepo := repository.NewUploadRepository(database.Pool)

	// ==================== SERVICES ====================
	authService := service.NewAuthService(authRepo, userRepo, cfg)
	userService := service.NewUserService(userRepo, followRepo)
	portfolioService := service.NewPortfolioService(portfolioRepo, tagRepo, userRepo)
	tagService := service.NewTagService(tagRepo)
	adminService := service.NewAdminService(majorRepo, yearRepo, classRepo, userRepo)

	// Upload service with MinIO
	uploadService, err := service.NewUploadService(cfg, uploadRepo, userRepo, portfolioRepo)
	if err != nil {
		log.Fatalf("Failed to create upload service: %v", err)
	}

	// Ensure MinIO bucket exists
	if err := uploadService.EnsureBucketExists(context.Background()); err != nil {
		log.Printf("Warning: Failed to ensure bucket exists: %v", err)
	}

	// ==================== FIBER APP ====================
	app := fiber.New(fiber.Config{
		AppName:       "Grafikarsa API",
		StrictRouting: false,
		CaseSensitive: false,
		ReadTimeout:   30 * time.Second,
		WriteTimeout:  30 * time.Second,
		IdleTimeout:   120 * time.Second,
		ErrorHandler:  customErrorHandler,
	})

	// Setup common middleware
	middleware.SetupCommon(app)

	// ==================== MIDDLEWARE ====================
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// ==================== HANDLERS ====================
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	portfolioHandler := handler.NewPortfolioHandler(portfolioService)
	uploadHandler := handler.NewUploadHandler(uploadService)
	adminHandler := handler.NewAdminHandler(adminService, userService, tagService, portfolioService)
	publicHandler := handler.NewPublicHandler(tagService, adminService)
	searchHandler := handler.NewSearchHandler(userService, portfolioService)

	// ==================== ROUTES ====================
	// API v1 routes
	api := app.Group("/api/v1")

	// Apply rate limiter to auth routes
	// api.Use("/auth", middleware.AuthRateLimiter())

	// Register handlers
	publicHandler.Register(api, authMiddleware)
	authHandler.Register(api, authMiddleware)
	userHandler.Register(api, authMiddleware)
	portfolioHandler.Register(api, authMiddleware)
	uploadHandler.Register(api, authMiddleware)
	adminHandler.Register(api, authMiddleware)
	searchHandler.Register(api, authMiddleware)

	// ==================== START SERVER ====================
	addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)

	// Graceful shutdown
	go func() {
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Printf("Server started on %s", addr)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// customErrorHandler handles uncaught errors
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error": fiber.Map{
			"code":    "INTERNAL_ERROR",
			"message": "Terjadi kesalahan pada server",
		},
	})
}
