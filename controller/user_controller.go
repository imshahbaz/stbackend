package controller

import (
	"backend/middleware"
	"backend/model"
	"backend/service"
	"context"
	"net/http"
	"time"

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
	authGroup := router.Group("/user")
	protected := authGroup.Group("/")
	protected.Use(middleware.AuthMiddleware(ctrl.isProduction))
	{
		protected.PATCH("/username", ctrl.UpdateUsername)
		protected.PATCH("/theme", ctrl.UpdateTheme)
	}
}

// UpdateUsername godoc
// @Summary      Update Username
// @Description  Updates the username and returns the updated user object
// @Tags         User
// @Accept       json
// @Produce      json
// @Param        update  body      model.UserDto  true  "Target Email and New Username"
// @Success      200     {object}  model.UserDto
// @Failure      400     {object}  map[string]string
// @Failure      404     {object}  map[string]string
// @Router       /user/username [patch]
func (ctrl *UserController) UpdateUsername(c *gin.Context) {
	var req model.UserDto

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	// 1. Call Service (Returns updated *model.User)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updatedUser, err := ctrl.userSvc.UpdateUsername(ctx, req.Email, req.Username)
	if err != nil {
		// Handle case where user might not exist or DB error
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found or update failed"})
		return
	}

	// 2. Return the DTO (React will use this to update its state)
	c.JSON(http.StatusOK, updatedUser.ToDto())
}

// UpdateTheme godoc
// @Summary      Update User Theme
// @Description  Updates the theme preference (LIGHT/DARK) for the authenticated user
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        request body      model.UpdateThemeRequest  true  "Theme Preference"
// @Success      200     {object}  model.Response
// @Failure      400     {object}  model.Response
// @Failure      401     {object}  model.Response
// @Failure      500     {object}  model.Response
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
			Error:   "Theme must be either LIGHT or DARK",
		})
		return
	}

	val, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.Response{
			Success: false,
			Error:   "Unauthorized: User",
		})
		return
	}

	user, ok := val.(model.UserDto)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.Response{
			Success: false,
			Error:   "Unauthorized: User",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := ctrl.userSvc.UpdateUserTheme(ctx, user.Email, req.Theme)

	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   "Something went wrong",
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "Theme synchronized",
		Data:    req.Theme,
	})

}
