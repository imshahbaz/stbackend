package middleware

import (
	"net/http"

	localCache "backend/cache"
	"backend/config"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
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
