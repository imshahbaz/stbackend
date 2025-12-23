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
