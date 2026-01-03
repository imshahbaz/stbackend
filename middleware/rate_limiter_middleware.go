package middleware

import (
	"net/http"
	"time"

	localCache "backend/cache"
	"backend/config"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

func RateLimiter(cfg *config.ConfigManager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !cfg.GetConfig().RateLimiter {
			ctx.Next()
			return
		}
		ip := ctx.ClientIP()

		var limiter *rate.Limiter
		if val, found := localCache.RateLimiterCache.Get(ip); found {
			limiter = val.(*rate.Limiter)
		} else {
			limiter = rate.NewLimiter(rate.Limit(5), 15)
			localCache.RateLimiterCache.Set(ip, limiter, cache.DefaultExpiration)
		}

		if !limiter.Allow() {
			ctx.Header("Retry-After", "5")

			ctx.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please wait 5 seconds before trying again.",
				"retry":   5,
			})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

func RecoveryMiddleware(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {

			log.Error().
				Interface("panic", err).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Msg("PANIC_RECOVERED")

			c.AbortWithStatusJSON(500, gin.H{
				"success": false,
				"message": "Internal server error",
				"error":   "unexpected_panic",
			})
		}
	}()
	c.Next()
}

func ZerologMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/api/health" || path == "/openapi.yaml" || path == "/service-worker.js" {
			c.Next()
			return
		}

		start := time.Now()
		query := c.Request.URL.RawQuery

		c.Next()
		latency := time.Since(start)

		log.Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Msg("HTTP Request")
	}
}
