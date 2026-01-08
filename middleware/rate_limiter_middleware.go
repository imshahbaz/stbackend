package middleware

import (
	"time"

	"backend/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/rs/zerolog/log"
)

var (
	skipPaths = map[string]bool{
		"/api/health":        true,
		"/openapi.yaml":      true,
		"/service-worker.js": true,
		"/favicon.ico":       true,
	}
)

func RateLimiter(cfg *config.ConfigManager) fiber.Handler {
	return limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool {
			return !cfg.GetConfig().RateLimiter
		},

		Max:        15,
		Expiration: 1 * time.Second,

		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},

		LimitReached: func(c *fiber.Ctx) error {
			c.Set("Retry-After", "1")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please wait 1 second.",
				"retry":   1,
			})
		},
	})
}

func RecoveryMiddleware(c *fiber.Ctx) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error().
				Interface("panic", err).
				Str("path", c.Path()).
				Str("method", c.Method()).
				Msg("PANIC_RECOVERED")

			c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Internal server error",
				"error":   "unexpected_panic",
			})
		}
	}()

	return c.Next()
}

func ZerologMiddleware() fiber.Handler {
	skipPaths := map[string]bool{
		"/health":  true,
		"/metrics": true,
	}

	return func(c *fiber.Ctx) error {
		path := c.Path()

		if skipPaths[path] {
			return c.Next()
		}

		start := time.Now()

		err := c.Next()

		latency := time.Since(start)
		query := c.Queries()

		log.Info().
			Str("method", c.Method()).
			Str("path", path).
			Interface("query", query).
			Int("status", c.Response().StatusCode()).
			Dur("latency", latency).
			Msg("HTTP Request")

		return err
	}
}
