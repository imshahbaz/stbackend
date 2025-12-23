package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"backend/auth"
	"backend/cache"
	"backend/config"
	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct {
	userSvc service.UserService
	cfg     *config.SystemConfigs
	otpSvc  service.OtpService
}

func NewAuthController(s service.UserService, cfg *config.SystemConfigs, otpSvc service.OtpService) *AuthController {
	return &AuthController{userSvc: s, cfg: cfg, otpSvc: otpSvc}
}

func (ctrl *AuthController) RegisterRoutes(router *gin.RouterGroup) {
	authGroup := router.Group("/auth")

	// 1. Public Routes
	authGroup.POST("/login", ctrl.Login)
	authGroup.POST("/signup", ctrl.Signup)
	authGroup.POST("/verify-otp", ctrl.VerifyOtp)

	// 2. Protected Routes (Apply middleware to this sub-group)
	protected := authGroup.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.PATCH("/username", ctrl.UpdateUsername)
		protected.POST("/logout", ctrl.Logout)
		protected.GET("/me", ctrl.GetMe)
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
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password"})
		return
	}

	// 3. Verify Bcrypt Password
	hashedFromDB := strings.TrimSpace(user.Password)
	if err := bcrypt.CompareHashAndPassword([]byte(hashedFromDB), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password"})
		return
	}

	response := user.ToDto()
	// 2. Generate the JWT
	token, err := auth.GenerateToken(response)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate token"})
		return
	}

	isProduction := ctrl.cfg.Config.Environment == "production"
	// 3. Set HttpOnly Cookie
	c.SetCookie(
		"auth_token", // name
		token,        // value
		1800,         // maxAge in seconds (30 mins)
		"/",          // path
		"",           // domain (empty for localhost)
		isProduction, // secure (set to TRUE in production for HTTPS)
		true,         // httpOnly (PREVENTS JAVASCRIPT ACCESS)
	)

	// 4. Return DTO (Response)
	// Because of json:"-", the password fields will be stripped automatically

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

// Logout godoc
// @Summary      Logout user
// @Description  Clears the authentication cookie to log out the user
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  map[string]string  "message: Logged out successfully"
// @Router       /auth/logout [post]
func (ctrl *AuthController) Logout(c *gin.Context) {
	// Set the cookie with a MaxAge of -1 to delete it instantly
	// In production, ensure 'secure' is set to true if using HTTPS
	isProduction := ctrl.cfg.Config.Environment == "production"
	c.SetCookie("auth_token", "", -1, "/", "", isProduction, true)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetMe godoc
// @Summary      Get current authenticated user
// @Description  Retrieves user details (email and role) from the session cookie
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  model.UserDto "User details successfully retrieved"
// @Failure      401  {object}  map[string]string    "Unauthorized: Session invalid or expired"
// @Router       /auth/me [get]
func (ctrl *AuthController) GetMe(c *gin.Context) {
	// 1. Extract the DTO using the helper we created earlier
	user, ok := middleware.GetUser(c)
	if !ok {
		// This case should rarely happen if the middleware is working correctly
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User session not found"})
		return
	}

	// 2. Return the DTO directly to React
	c.JSON(http.StatusOK, user)
}

// Signup handles user registration and caches pending data locally
// @Summary      User Signup
// @Description  Stores user data in local memory for 5 minutes and sends an OTP.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        user  body      model.UserDto  true  "User Registration Details"
// @Success      200   {object}  model.MessageResponse
// @Failure      400   {object}  map[string]string
// @Router       /auth/signup [post]
func (ctrl *AuthController) Signup(c *gin.Context) {
	var user model.UserDto

	// 1. Validation
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// 2. Keep in Cache for 5 mins
	// We use the email as the key to retrieve the data during OTP verification
	cache.PendingUserCache.Set(user.Email, user, 5*time.Minute)

	// 3. Send OTP
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := ctrl.otpSvc.SendSignUpOtp(ctx, user); err != nil {

		if errors.Is(err, service.ErrDuplicateOtp) {
			c.JSON(http.StatusConflict, model.MessageResponse{
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong please try again later"})
		return
	}

	// 4. Return Success Response
	c.JSON(http.StatusOK, model.MessageResponse{
		OtpSent: true,
		Message: "Otp sent successfully to " + user.Email,
	})
}

// VerifyOtp handles OTP validation and final user creation
// @Summary      Verify OTP and Create User
// @Description  Validates the OTP from cache. If valid, creates the user in the database and returns a JWT.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      model.VerifyOtpRequest  true  "OTP Verification Details"
// @Success      201      {object}  model.MessageResponse  "User created successfully"
// @Failure      400      {object}  model.MessageResponse  "Invalid OTP or session expired"
// @Router       /auth/verify-otp [post]
func (ctrl *AuthController) VerifyOtp(c *gin.Context) {
	var req model.VerifyOtpRequest

	// 1. Bind JSON Input
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.MessageResponse{Message: "Invalid request format"})
		return
	}

	// 2. Retrieve Pending User from go-cache
	// We use the email from the request to find the cached data
	val, found := cache.PendingUserCache.Get(req.Email)
	if !found {
		c.JSON(http.StatusBadRequest, model.MessageResponse{
			Message: "No pending signup found or session expired. Please start over.",
		})
		return
	}

	// 3. Verify OTP Logic
	// Assuming otpService.Verify returns an error if invalid
	if match, err := ctrl.otpSvc.VerifyOtp(req.Email, req.Otp); err != nil || !match {
		c.JSON(http.StatusBadRequest, model.MessageResponse{Message: "Invalid or expired OTP"})
		return
	}

	// 4. Create User in Database
	// We pass the data we recovered from the cache
	pendingDto := val.(model.UserDto)
	_, err := ctrl.userSvc.CreateUser(c.Request.Context(), pendingDto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.MessageResponse{Message: "Failed to create user"})
		return
	}

	// 5. Cleanup Cache
	cache.PendingUserCache.Delete(req.Email)

	// 6. Return Success (In a JWT flow, you'd generate the token here)
	c.JSON(http.StatusCreated, model.MessageResponse{
		Message: "Signup successful! You can now login.",
	})
}
