package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/grafikarsa/backend/internal/auth"
	"github.com/grafikarsa/backend/internal/config"
	"github.com/grafikarsa/backend/internal/database"
	"github.com/grafikarsa/backend/internal/handler"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
	"github.com/grafikarsa/backend/internal/service"
	"github.com/grafikarsa/backend/internal/storage"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize MinIO client
	minioClient, err := storage.NewMinIOClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}

	// Initialize JWT service
	jwtService := auth.NewJWTService(cfg)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	authRepo := repository.NewAuthRepository(db)
	portfolioRepo := repository.NewPortfolioRepository(db)
	followRepo := repository.NewFollowRepository(db)
	adminRepo := repository.NewAdminRepository(db)
	feedbackRepo := repository.NewFeedbackRepository(db)
	assessmentRepo := repository.NewAssessmentRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	viewRepo := repository.NewViewRepository(db)
	interestRepo := repository.NewInterestRepository(db)
	feedRepo := repository.NewFeedRepository(db)
	changelogRepo := repository.NewChangelogRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	dmRepo := repository.NewDMRepository(db)

	// Initialize services
	notificationService := service.NewNotificationService(notificationRepo)
	feedService := service.NewFeedService(portfolioRepo, followRepo, viewRepo, interestRepo)
	commentService := service.NewCommentService(commentRepo, userRepo, portfolioRepo, notificationRepo)
	commentService.SetNotificationService(notificationService)
	dmService := service.NewDMService(dmRepo, userRepo, followRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(userRepo, authRepo, jwtService)
	userHandler := handler.NewUserHandler(userRepo, followRepo, notificationService)
	profileHandler := handler.NewProfileHandler(userRepo, adminRepo)
	portfolioHandler := handler.NewPortfolioHandler(portfolioRepo, userRepo, viewRepo, interestRepo, notificationService)
	contentBlockHandler := handler.NewContentBlockHandler(portfolioRepo)
	adminHandler := handler.NewAdminHandler(adminRepo, userRepo, portfolioRepo, notificationService)
	uploadHandler := handler.NewUploadHandler(minioClient, userRepo, portfolioRepo)
	tagHandler := handler.NewTagHandler(adminRepo)
	publicHandler := handler.NewPublicHandler(adminRepo, userRepo)
	feedHandler := handler.NewFeedHandler(feedRepo, feedService, interestRepo, userRepo)
	searchHandler := handler.NewSearchHandler(userRepo, portfolioRepo)
	feedbackHandler := handler.NewFeedbackHandler(feedbackRepo, userRepo, notificationService)
	assessmentHandler := handler.NewAssessmentHandler(assessmentRepo, portfolioRepo)
	notificationHandler := handler.NewNotificationHandler(notificationRepo, userRepo, followRepo)
	importHandler := handler.NewImportHandler(adminRepo, userRepo)
	changelogHandler := handler.NewChangelogHandler(changelogRepo, notificationService, userRepo)
	commentHandler := handler.NewCommentHandler(commentService)
	dmHandler := handler.NewDMHandler(dmService)
	wsHandler := handler.NewWebSocketHandler()

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtService, db)
	capMiddleware := middleware.NewCapabilityMiddleware(adminRepo)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "INTERNAL_ERROR",
					"message": err.Error(),
				},
			})
		},
	})

	// 1. CORS MUST be the very first middleware for proper preflight handling
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.CORS.Origins, ","),
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With,Cache-Control,Pragma,Service-Worker",
		AllowCredentials: true,
		MaxAge:           3600, // Cache preflight for 1 hour
	}))

	// 2. Standard middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// Rate Limiting
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		Next: func(c *fiber.Ctx) bool {
			return c.Method() == "OPTIONS"
		},
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "TOO_MANY_REQUESTS",
					"message": "Terlalu banyak permintaan. Silakan coba lagi nanti.",
				},
			})
		},
	}))

	// API v1 routes
	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Auth routes
	authRateLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		Next: func(c *fiber.Ctx) bool {
			return c.Method() == "OPTIONS"
		},
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "TOO_MANY_REQUESTS",
					"message": "Terlalu banyak percobaan login. Silakan coba lagi dalam satu menit.",
				},
			})
		},
	})

	authRoutes := api.Group("/auth")
	authRoutes.Post("/login", authRateLimiter, authHandler.Login)
	authRoutes.Post("/refresh", authRateLimiter, authHandler.Refresh)
	authRoutes.Post("/logout", authMiddleware.Required(), authHandler.Logout)
	authRoutes.Post("/logout-all", authMiddleware.Required(), authHandler.LogoutAll)
	authRoutes.Get("/sessions", authMiddleware.Required(), authHandler.GetSessions)
	authRoutes.Delete("/sessions/:session_id", authMiddleware.Required(), authHandler.DeleteSession)

	// User routes
	userRoutes := api.Group("/users")
	userRoutes.Get("/", authMiddleware.Optional(), userHandler.List)
	userRoutes.Get("/:username", authMiddleware.Optional(), userHandler.GetByUsername)
	userRoutes.Get("/:username/followers", authMiddleware.Optional(), userHandler.GetFollowers)
	userRoutes.Get("/:username/following", authMiddleware.Optional(), userHandler.GetFollowing)
	userRoutes.Post("/:username/follow", authMiddleware.Required(), userHandler.Follow)
	userRoutes.Delete("/:username/follow", authMiddleware.Required(), userHandler.Unfollow)

	// Profile routes (me)
	api.Get("/me", authMiddleware.Required(), profileHandler.GetMe)
	api.Patch("/me", authMiddleware.Required(), profileHandler.UpdateMe)
	api.Patch("/me/password", authMiddleware.Required(), profileHandler.UpdatePassword)
	api.Put("/me/social-links", authMiddleware.Required(), profileHandler.UpdateSocialLinks)
	api.Get("/me/check-username", authMiddleware.Required(), profileHandler.CheckUsername)
	api.Get("/me/portfolios", authMiddleware.Required(), portfolioHandler.GetMyPortfolios)

	// Portfolio routes
	portfolioRoutes := api.Group("/portfolios")
	portfolioRoutes.Get("/", authMiddleware.Optional(), portfolioHandler.List)
	portfolioRoutes.Post("/", authMiddleware.Required(), portfolioHandler.Create)
	portfolioRoutes.Get("/:slug", authMiddleware.Optional(), portfolioHandler.GetBySlug)
	portfolioRoutes.Get("/id/:id", authMiddleware.Required(), portfolioHandler.GetByID)
	portfolioRoutes.Patch("/:id", authMiddleware.Required(), portfolioHandler.Update)
	portfolioRoutes.Delete("/:id", authMiddleware.Required(), portfolioHandler.Delete)
	portfolioRoutes.Post("/:id/submit", authMiddleware.Required(), portfolioHandler.Submit)
	portfolioRoutes.Post("/:id/archive", authMiddleware.Required(), portfolioHandler.Archive)
	portfolioRoutes.Post("/:id/unarchive", authMiddleware.Required(), portfolioHandler.Unarchive)
	portfolioRoutes.Post("/:id/like", authMiddleware.Required(), portfolioHandler.Like)
	portfolioRoutes.Post("/:id/like", authMiddleware.Required(), portfolioHandler.Like)
	portfolioRoutes.Delete("/:id/like", authMiddleware.Required(), portfolioHandler.Unlike)

	// Comment Routes (nested under portfolios)
	portfolioRoutes.Post("/:portfolio_id/comments", authMiddleware.Required(), commentHandler.Create)
	portfolioRoutes.Get("/:id/comments", authMiddleware.Optional(), commentHandler.GetByPortfolioID)

	// Global Comment Routes
	commentRoutes := api.Group("/comments", authMiddleware.Required())
	commentRoutes.Delete("/:id", commentHandler.Delete)

	// Content block routes
	portfolioRoutes.Post("/:portfolio_id/blocks", authMiddleware.Required(), contentBlockHandler.Create)
	portfolioRoutes.Patch("/:portfolio_id/blocks/:block_id", authMiddleware.Required(), contentBlockHandler.Update)
	portfolioRoutes.Put("/:portfolio_id/blocks/reorder", authMiddleware.Required(), contentBlockHandler.Reorder)
	portfolioRoutes.Delete("/:portfolio_id/blocks/:block_id", authMiddleware.Required(), contentBlockHandler.Delete)

	// Tags routes (public)
	api.Get("/tags", tagHandler.List)

	// Public routes
	api.Get("/jurusan", publicHandler.ListJurusan)
	api.Get("/kelas", publicHandler.ListKelas)
	api.Get("/series", publicHandler.ListSeries)
	api.Get("/series/:id", publicHandler.GetSeries)
	api.Get("/top-students", publicHandler.GetTopStudents)
	api.Get("/top-projects", publicHandler.GetTopProjects)

	// Feed routes
	api.Get("/feed", authMiddleware.Optional(), feedHandler.GetFeed)
	api.Get("/feed/preferences", authMiddleware.Required(), feedHandler.GetFeedPreferences)

	// Changelog routes (public)
	changelogRoutes := api.Group("/changelogs")
	changelogRoutes.Get("/", authMiddleware.Optional(), changelogHandler.List)
	changelogRoutes.Get("/latest", authMiddleware.Optional(), changelogHandler.GetLatest)
	changelogRoutes.Get("/unread-count", authMiddleware.Required(), changelogHandler.GetUnreadCount)
	changelogRoutes.Post("/mark-all-read", authMiddleware.Required(), changelogHandler.MarkAllAsRead)
	changelogRoutes.Get("/:id", authMiddleware.Optional(), changelogHandler.GetByID)
	changelogRoutes.Post("/:id/mark-read", authMiddleware.Required(), changelogHandler.MarkAsRead)
	api.Put("/feed/preferences", authMiddleware.Required(), feedHandler.UpdateFeedPreferences)

	// Notification routes
	notifRoutes := api.Group("/notifications", authMiddleware.Required())
	notifRoutes.Get("/", notificationHandler.List)
	notifRoutes.Get("/count", notificationHandler.Count)
	notifRoutes.Patch("/:id/read", notificationHandler.MarkAsRead)
	notifRoutes.Post("/read-all", notificationHandler.MarkAllAsRead)
	notifRoutes.Delete("/:id", notificationHandler.Delete)

	// Search routes
	searchRoutes := api.Group("/search")
	searchRoutes.Get("/users", authMiddleware.Optional(), searchHandler.SearchUsers)
	searchRoutes.Get("/portfolios", authMiddleware.Optional(), searchHandler.SearchPortfolios)

	// Upload routes
	uploadRoutes := api.Group("/uploads")
	uploadRoutes.Post("/presign", authMiddleware.Required(), uploadHandler.Presign)
	uploadRoutes.Post("/confirm", authMiddleware.Required(), uploadHandler.Confirm)
	uploadRoutes.Delete("/*", authMiddleware.Required(), uploadHandler.Delete)
	uploadRoutes.Get("/presign-view", authMiddleware.Required(), uploadHandler.PresignView)

	// Admin routes - base group with auth required
	adminRoutes := api.Group("/admin", authMiddleware.Required())

	// Admin - Dashboard (requires dashboard capability)
	adminRoutes.Get("/dashboard/stats", capMiddleware.RequireCapability("dashboard"), adminHandler.GetDashboardStats)

	// Admin - Jurusan (requires majors capability)
	adminRoutes.Get("/jurusan", capMiddleware.RequireCapability("majors"), adminHandler.ListJurusan)
	adminRoutes.Post("/jurusan", capMiddleware.RequireCapability("majors"), adminHandler.CreateJurusan)
	adminRoutes.Patch("/jurusan/:id", capMiddleware.RequireCapability("majors"), adminHandler.UpdateJurusan)
	adminRoutes.Delete("/jurusan/:id", capMiddleware.RequireCapability("majors"), adminHandler.DeleteJurusan)

	// Admin - Tahun Ajaran (requires academic_years capability)
	adminRoutes.Get("/tahun-ajaran", capMiddleware.RequireCapability("academic_years"), adminHandler.ListTahunAjaran)
	adminRoutes.Post("/tahun-ajaran", capMiddleware.RequireCapability("academic_years"), adminHandler.CreateTahunAjaran)
	adminRoutes.Patch("/tahun-ajaran/:id", capMiddleware.RequireCapability("academic_years"), adminHandler.UpdateTahunAjaran)
	adminRoutes.Delete("/tahun-ajaran/:id", capMiddleware.RequireCapability("academic_years"), adminHandler.DeleteTahunAjaran)

	// Admin - Kelas (requires classes capability)
	adminRoutes.Get("/kelas", capMiddleware.RequireCapability("classes"), adminHandler.ListKelas)
	adminRoutes.Post("/kelas", capMiddleware.RequireCapability("classes"), adminHandler.CreateKelas)
	adminRoutes.Patch("/kelas/:id", capMiddleware.RequireCapability("classes"), adminHandler.UpdateKelas)
	adminRoutes.Delete("/kelas/:id", capMiddleware.RequireCapability("classes"), adminHandler.DeleteKelas)
	adminRoutes.Get("/kelas/:id/students", capMiddleware.RequireCapability("classes"), adminHandler.GetKelasStudents)

	// Admin - Tags (requires tags capability)
	adminRoutes.Get("/tags", capMiddleware.RequireCapability("tags"), adminHandler.ListTags)
	adminRoutes.Post("/tags", capMiddleware.RequireCapability("tags"), adminHandler.CreateTag)
	adminRoutes.Patch("/tags/:id", capMiddleware.RequireCapability("tags"), adminHandler.UpdateTag)
	adminRoutes.Delete("/tags/:id", capMiddleware.RequireCapability("tags"), adminHandler.DeleteTag)

	// Admin - Series (requires series capability)
	adminRoutes.Get("/series", capMiddleware.RequireCapability("series"), adminHandler.ListSeries)
	adminRoutes.Get("/series/:id", capMiddleware.RequireCapability("series"), adminHandler.GetSeries)
	adminRoutes.Post("/series", capMiddleware.RequireCapability("series"), adminHandler.CreateSeries)
	adminRoutes.Patch("/series/:id", capMiddleware.RequireCapability("series"), adminHandler.UpdateSeries)
	adminRoutes.Delete("/series/:id", capMiddleware.RequireCapability("series"), adminHandler.DeleteSeries)
	adminRoutes.Get("/series/:id/export/preview", capMiddleware.RequireCapability("series"), adminHandler.GetSeriesExportPreview)
	adminRoutes.Get("/series/:id/export", capMiddleware.RequireCapability("series"), adminHandler.GetSeriesExportData)

	// Admin - Users (requires users capability)
	adminRoutes.Get("/users", capMiddleware.RequireCapability("users"), adminHandler.ListUsers)
	adminRoutes.Get("/users/check-username", capMiddleware.RequireCapability("users"), adminHandler.CheckUsername)
	adminRoutes.Get("/users/check-email", capMiddleware.RequireCapability("users"), adminHandler.CheckEmail)
	adminRoutes.Post("/users", capMiddleware.RequireCapability("users"), adminHandler.CreateUser)
	adminRoutes.Get("/users/:id", capMiddleware.RequireCapability("users"), adminHandler.GetUser)
	adminRoutes.Patch("/users/:id", capMiddleware.RequireCapability("users"), adminHandler.UpdateUser)
	adminRoutes.Patch("/users/:id/password", capMiddleware.RequireCapability("users"), adminHandler.ResetUserPassword)
	adminRoutes.Delete("/users/:id", capMiddleware.RequireCapability("users"), adminHandler.DeleteUser)
	adminRoutes.Post("/users/:id/deactivate", capMiddleware.RequireCapability("users"), adminHandler.DeactivateUser)
	adminRoutes.Post("/users/:id/activate", capMiddleware.RequireCapability("users"), adminHandler.ActivateUser)

	// Admin - User Special Roles (requires users capability)
	adminRoutes.Get("/users/:id/special-roles", capMiddleware.RequireCapability("users"), adminHandler.GetUserSpecialRoles)
	adminRoutes.Put("/users/:id/special-roles", capMiddleware.RequireCapability("users"), adminHandler.UpdateUserSpecialRoles)

	// Admin - Import Students (requires users capability)
	adminRoutes.Post("/import/students", capMiddleware.RequireCapability("users"), importHandler.ImportStudents)
	adminRoutes.Get("/import/students/template", capMiddleware.RequireCapability("users"), importHandler.DownloadTemplate)

	// Admin - Portfolios (requires portfolios capability)
	adminRoutes.Get("/portfolios", capMiddleware.RequireCapability("portfolios"), adminHandler.ListAllPortfolios)
	adminRoutes.Get("/portfolios/pending", capMiddleware.RequireCapability("moderation"), adminHandler.ListPendingPortfolios)
	adminRoutes.Get("/portfolios/:id", capMiddleware.RequireCapability("portfolios"), adminHandler.GetPortfolio)
	adminRoutes.Patch("/portfolios/:id", capMiddleware.RequireCapability("portfolios"), adminHandler.UpdatePortfolio)
	adminRoutes.Delete("/portfolios/:id", capMiddleware.RequireCapability("portfolios"), adminHandler.DeletePortfolio)
	adminRoutes.Post("/portfolios/:id/approve", capMiddleware.RequireCapability("moderation"), adminHandler.ApprovePortfolio)
	adminRoutes.Post("/portfolios/:id/reject", capMiddleware.RequireCapability("moderation"), adminHandler.RejectPortfolio)

	// Admin - Feedback (requires feedback capability)
	adminRoutes.Get("/feedback", capMiddleware.RequireCapability("feedback"), feedbackHandler.AdminListFeedback)
	adminRoutes.Get("/feedback/stats", capMiddleware.RequireCapability("feedback"), feedbackHandler.AdminGetFeedbackStats)
	adminRoutes.Get("/feedback/:id", capMiddleware.RequireCapability("feedback"), feedbackHandler.AdminGetFeedback)
	adminRoutes.Patch("/feedback/:id", capMiddleware.RequireCapability("feedback"), feedbackHandler.AdminUpdateFeedback)
	adminRoutes.Delete("/feedback/:id", capMiddleware.RequireCapability("feedback"), feedbackHandler.AdminDeleteFeedback)

	// Admin - Changelogs (requires changelog capability)
	adminRoutes.Get("/changelogs", capMiddleware.RequireCapability("changelog"), changelogHandler.AdminList)
	adminRoutes.Get("/changelogs/:id", capMiddleware.RequireCapability("changelog"), changelogHandler.AdminGetByID)
	adminRoutes.Post("/changelogs", capMiddleware.RequireCapability("changelog"), changelogHandler.Create)
	adminRoutes.Patch("/changelogs/:id", capMiddleware.RequireCapability("changelog"), changelogHandler.Update)
	adminRoutes.Delete("/changelogs/:id", capMiddleware.RequireCapability("changelog"), changelogHandler.Delete)
	adminRoutes.Post("/changelogs/:id/publish", capMiddleware.RequireCapability("changelog"), changelogHandler.Publish)
	adminRoutes.Post("/changelogs/:id/unpublish", capMiddleware.RequireCapability("changelog"), changelogHandler.Unpublish)

	// Admin - Assessment Metrics (requires assessment_metrics capability)
	adminRoutes.Get("/assessment-metrics", capMiddleware.RequireCapability("assessment_metrics"), assessmentHandler.ListMetrics)
	adminRoutes.Post("/assessment-metrics", capMiddleware.RequireCapability("assessment_metrics"), assessmentHandler.CreateMetric)
	adminRoutes.Put("/assessment-metrics/reorder", capMiddleware.RequireCapability("assessment_metrics"), assessmentHandler.ReorderMetrics)
	adminRoutes.Patch("/assessment-metrics/:id", capMiddleware.RequireCapability("assessment_metrics"), assessmentHandler.UpdateMetric)
	adminRoutes.Delete("/assessment-metrics/:id", capMiddleware.RequireCapability("assessment_metrics"), assessmentHandler.DeleteMetric)

	// Admin - Portfolio Assessments (requires assessments capability)
	adminRoutes.Get("/assessments", capMiddleware.RequireCapability("assessments"), assessmentHandler.ListPortfoliosForAssessment)
	adminRoutes.Get("/assessments/stats", capMiddleware.RequireCapability("assessments"), assessmentHandler.GetAssessmentStats)
	adminRoutes.Get("/assessments/:portfolio_id", capMiddleware.RequireCapability("assessments"), assessmentHandler.GetAssessment)
	adminRoutes.Post("/assessments/:portfolio_id", capMiddleware.RequireCapability("assessments"), assessmentHandler.CreateOrUpdateAssessment)
	adminRoutes.Delete("/assessments/:portfolio_id", capMiddleware.RequireCapability("assessments"), assessmentHandler.DeleteAssessment)

	// Admin - Special Roles (requires special_roles capability - admin only by default)
	adminRoutes.Get("/special-roles", capMiddleware.RequireCapability("special_roles"), adminHandler.ListSpecialRoles)
	adminRoutes.Get("/special-roles/active", capMiddleware.RequireCapability("special_roles"), adminHandler.GetActiveSpecialRoles)
	adminRoutes.Get("/special-roles/capabilities", capMiddleware.RequireCapability("special_roles"), adminHandler.GetCapabilities)
	adminRoutes.Post("/special-roles", capMiddleware.RequireCapability("special_roles"), adminHandler.CreateSpecialRole)
	adminRoutes.Get("/special-roles/:id", capMiddleware.RequireCapability("special_roles"), adminHandler.GetSpecialRole)
	adminRoutes.Patch("/special-roles/:id", capMiddleware.RequireCapability("special_roles"), adminHandler.UpdateSpecialRole)
	adminRoutes.Delete("/special-roles/:id", capMiddleware.RequireCapability("special_roles"), adminHandler.DeleteSpecialRole)
	adminRoutes.Post("/special-roles/:id/users", capMiddleware.RequireCapability("special_roles"), adminHandler.AssignUsersToRole)
	adminRoutes.Delete("/special-roles/:id/users/:userId", capMiddleware.RequireCapability("special_roles"), adminHandler.RemoveUserFromRole)

	// Public Feedback route (auth optional)
	api.Post("/feedback", authMiddleware.Optional(), feedbackHandler.CreateFeedback)

	// Direct Messaging routes
	conversationRoutes := api.Group("/conversations", authMiddleware.Required())
	conversationRoutes.Get("/", dmHandler.ListConversations)
	conversationRoutes.Post("/", dmHandler.StartConversation)
	conversationRoutes.Get("/:id", dmHandler.GetConversation)
	conversationRoutes.Post("/:id/archive", dmHandler.ArchiveConversation)
	conversationRoutes.Delete("/:id/archive", dmHandler.UnarchiveConversation)
	conversationRoutes.Post("/:id/mute", dmHandler.MuteConversation)
	conversationRoutes.Delete("/:id/mute", dmHandler.UnmuteConversation)
	conversationRoutes.Post("/:id/read", dmHandler.MarkAsRead)
	conversationRoutes.Get("/:id/messages", dmHandler.GetMessages)
	conversationRoutes.Post("/:id/messages", dmHandler.SendMessage)

	// Message routes
	messageRoutes := api.Group("/messages", authMiddleware.Required())
	messageRoutes.Delete("/:id", dmHandler.DeleteMessage)
	messageRoutes.Post("/:id/reactions", dmHandler.AddReaction)
	messageRoutes.Delete("/:id/reactions/:emoji", dmHandler.RemoveReaction)

	// DM Settings & Block routes
	dmRoutes := api.Group("/dm", authMiddleware.Required())
	dmRoutes.Get("/settings", dmHandler.GetDMSettings)
	dmRoutes.Patch("/settings", dmHandler.UpdateDMSettings)
	dmRoutes.Post("/block/:userId", dmHandler.BlockUser)
	dmRoutes.Delete("/block/:userId", dmHandler.UnblockUser)
	dmRoutes.Get("/blocked", dmHandler.GetBlockedUsers)
	dmRoutes.Get("/streaks", dmHandler.GetChatStreaks)

	// WebSocket for real-time DM
	app.Use("/api/v1/ws/chat", wsHandler.WebSocketUpgrade(authMiddleware))
	app.Get("/api/v1/ws/chat", websocket.New(wsHandler.HandleWebSocket))

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Gracefully shutting down...")
		_ = app.Shutdown()
	}()

	// Start server
	port := cfg.App.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on 0.0.0.0:%s", port)
	if err := app.Listen("0.0.0.0:" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
