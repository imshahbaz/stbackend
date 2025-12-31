package middleware

import (
	"backend/config"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS(cfg *config.ConfigManager) gin.HandlerFunc {
	return cors.New(cors.Config{
		// 1. Specify your exact frontend origin (avoid "*" when using credentials)
		AllowOrigins: cfg.GetConfig().FrontendUrls,

		// 2. Methods your React app is allowed to use
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},

		// 3. Headers allowed in requests (important for Auth and JSON)
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		},

		// 4. Headers the browser is allowed to read from the response
		ExposeHeaders: []string{"Content-Length"},

		// 5. CRITICAL: Must be true to allow HttpOnly Cookies/JWTs to be sent
		AllowCredentials: true,

		// 6. How long the browser should cache the CORS preflight (OPTIONS) response
		MaxAge: 12 * time.Hour,
	})
}
