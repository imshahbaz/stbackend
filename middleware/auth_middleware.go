package middleware

import (
	"net/http"
	"time"

	"backend/auth"
	"backend/model"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("auth_token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Success: false,
				Error:   "Unauthorized",
			})
			return
		}

		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Success: false,
				Error:   "Invalid session",
			})
			return
		}

		if time.Until(claims.ExpiresAt.Time) < 15*time.Minute {
			if isProduction {
				c.SetSameSite(http.SameSiteNoneMode)
			}
			newToken, _ := auth.GenerateToken(claims.User)
			c.SetCookie("auth_token", newToken, 1800, "/", "", isProduction, true)
		}

		c.Set("user", claims.User)
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Using a helper makes the middleware much shorter
		user, ok := GetUser(c)

		if !ok || user.Role != model.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
				Success: false,
				Error:   "Forbidden: Admin access required",
			})
			return
		}

		c.Next()
	}
}

// GetUser is a helper to extract the DTO from the context safely
func GetUser(c *gin.Context) (model.UserDto, bool) {
	// 1. Pull the value from the context using the key set in AuthMiddleware
	val, exists := c.Get("user")
	if !exists {
		return model.UserDto{}, false
	}

	// 2. Type assertion: Convert 'any' back to your specific DTO struct
	user, ok := val.(model.UserDto)
	return user, ok
}
