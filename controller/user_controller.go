package controller

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"backend/cache"
	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userSvc      service.UserService
	isProduction bool
}

func NewUserController(s service.UserService, isProduction bool) *UserController {
	return &UserController{userSvc: s, isProduction: isProduction}
}

func (ctrl *UserController) RegisterRoutes(router *gin.RouterGroup) {
	userGroup := router.Group("/user")
	userGroup.Use(middleware.AuthMiddleware(ctrl.isProduction))
	{
		userGroup.PATCH("/username", ctrl.UpdateUsername)
		userGroup.PATCH("/theme", ctrl.UpdateTheme)
	}
}

// UpdateUsername godoc
// @Summary      Update Username
// @Description  Updates the username and invalidates the auth cache
// @Tags         User
// @Accept       json
// @Produce      json
// @Param        update  body      model.UserDto  true  "Target Email and New Username"
// @Success      200     {object}  model.Response
// @Failure      400     {object}  model.Response
// @Failure      401     {object}  model.Response
// @Router       /user/username [patch]
func (ctrl *UserController) UpdateUsername(c *gin.Context) {
	var req model.UserDto
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   "Invalid request payload",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	_, err := ctrl.userSvc.UpdateUsername(ctx, req.UserID, req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   "Failed to update username",
		})
		return
	}

	// Cache Invalidation: Force refresh on next GetMe call
	cache.UserAuthCache.Delete(strconv.FormatInt(req.UserID, 10))

	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "Username updated successfully",
	})
}

// UpdateTheme godoc
// @Summary      Update User Theme
// @Description  Updates preference (LIGHT/DARK) for the authenticated user
// @Tags         User
// @Accept       json
// @Produce      json
// @Param        request body      model.UpdateThemeRequest  true  "Theme Preference"
// @Success      200     {object}  model.Response
// @Failure      400     {object}  model.Response
// @Router       /user/theme [patch]
func (ctrl *UserController) UpdateTheme(c *gin.Context) {
	var req model.UpdateThemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	if req.Theme != model.ThemeLight && req.Theme != model.ThemeDark {
		c.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   "Invalid theme: must be LIGHT or DARK",
		})
		return
	}

	// Extract user from context (set by AuthMiddleware)
	val, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.Response{
			Success: false,
			Error:   "User session not found",
		})
		return
	}

	userDto := val.(model.UserDto)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if _, err := ctrl.userSvc.UpdateUserTheme(ctx, userDto.UserID, req.Theme); err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   "Internal server error",
		})
		return
	}

	// Clear cache so that subsequent requests reflect the new theme
	cache.UserAuthCache.Delete(strconv.FormatInt(userDto.UserID, 10))

	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "Theme synchronized",
		Data:    req.Theme,
	})
}
