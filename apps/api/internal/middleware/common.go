package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// SetupCommon sets up common middleware
func SetupCommon(app *fiber.App) {
	// Recover from panics
	app.Use(recover.New())

	// Logger
	app.Use(logger.New(logger.Config{
		Format:     "${time} ${status} ${method} ${path} ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Local",
	}))

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: false,
		MaxAge:           86400,
	}))
}

// RateLimiter creates a rate limiter middleware
func RateLimiter(max int, duration time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: duration,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use IP address as key, prefer X-Forwarded-For if behind proxy
			if xff := c.Get("X-Forwarded-For"); xff != "" {
				return xff
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Terlalu banyak permintaan. Silakan coba lagi nanti.",
				},
			})
		},
	})
}

// AuthRateLimiter creates a stricter rate limiter for auth endpoints
func AuthRateLimiter() fiber.Handler {
	return RateLimiter(5, time.Minute)
}

// GeneralRateLimiter creates a general rate limiter
func GeneralRateLimiter() fiber.Handler {
	return RateLimiter(100, time.Minute)
}
