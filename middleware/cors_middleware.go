package middleware

import (
	"backend/config"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func CORS(cfg *config.ConfigManager) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: strings.Join(cfg.GetConfig().FrontendUrls, ","),

		AllowMethods: "GET,POST,PUT,PATCH,DELETE,HEAD,OPTIONS",

		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Requested-With",

		ExposeHeaders: "Content-Length",

		AllowCredentials: true,

		MaxAge: int((12 * time.Hour).Seconds()),
	})
}
