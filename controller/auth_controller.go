package controller

import (
	"net/http"
	"strings"

	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct {
	userSvc service.UserService
}

func NewAuthController(s service.UserService) *AuthController {
	return &AuthController{userSvc: s}
}

func (ctrl *AuthController) RegisterRoutes(router *gin.RouterGroup) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", ctrl.Login)
		authGroup.PATCH("/username", ctrl.UpdateUsername)
	}
}

// Login godoc
// @Summary      User Login
// @Description  Authenticates user and returns user details without password
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        login  body      model.UserDto  true  "Login Credentials (only email/password required)"
// @Success      200    {object}  model.UserDto  "Login successful (Passwords omitted)"
// @Failure      401    {object}  map[string]string "Unauthorized"
// @Router       /auth/login [post]
func (ctrl *AuthController) Login(c *gin.Context) {
	var req model.UserDto

	// 1. Bind JSON (Request)
	// Even though json:"-" is on passwords, Gin's ShouldBindJSON
	// will still map them if the JSON keys match "password"
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. Fetch User from DB
	user, err := ctrl.userSvc.GetUser(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// 3. Verify Bcrypt Password
	hashedFromDB := strings.TrimSpace(user.Password)
	if err := bcrypt.CompareHashAndPassword([]byte(hashedFromDB), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// 4. Return DTO (Response)
	// Because of json:"-", the password fields will be stripped automatically
	response := user.ToDto()
	c.JSON(http.StatusOK, response)
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
// @Router       /auth/username [patch]
func (ctrl *AuthController) UpdateUsername(c *gin.Context) {
	var req model.UserDto

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	// 1. Call Service (Returns updated *model.User)
	updatedUser, err := ctrl.userSvc.UpdateUsername(c.Request.Context(), req.Email, req.Username)
	if err != nil {
		// Handle case where user might not exist or DB error
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found or update failed"})
		return
	}

	// 2. Return the DTO (React will use this to update its state)
	c.JSON(http.StatusOK, updatedUser.ToDto())
}
