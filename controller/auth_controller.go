package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"backend/auth"
	localCache "backend/cache"
	"backend/config"
	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/patrickmn/go-cache"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct {
	userSvc      service.UserService
	cfgManager   *config.ConfigManager
	otpSvc       service.OtpService
	isProduction bool
	restyClient  *resty.Client
}

func NewAuthController(s service.UserService, cfgManager *config.ConfigManager,
	otpSvc service.OtpService, isProduction bool) *AuthController {
	return &AuthController{
		userSvc:      s,
		cfgManager:   cfgManager,
		otpSvc:       otpSvc,
		isProduction: isProduction,
		restyClient:  resty.New().SetTimeout(10 * time.Second),
	}
}

func (ctrl *AuthController) RegisterRoutes(router *gin.RouterGroup) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", ctrl.Login)
		authGroup.POST("/signup", ctrl.Signup)
		authGroup.POST("/verify-otp", ctrl.VerifyOtp)

		protected := authGroup.Group("/")
		protected.Use(middleware.AuthMiddleware(ctrl.isProduction))
		{
			protected.POST("/logout", ctrl.Logout)
			protected.GET("/me", ctrl.GetMe)
		}

		trueCallerGrp := authGroup.Group("/truecaller")
		{
			trueCallerGrp.POST("", ctrl.TrueCallerCallBack)
			trueCallerGrp.GET("/status/:requestId", ctrl.TrueCallerStatus)
		}
	}
}

// Login godoc
// @Summary      User Login
// @Description  Authenticates user via HttpOnly cookie and JWT
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        login  body      model.UserDto  true  "Login Credentials"
// @Success      200    {object}  model.UserDto
// @Failure      401    {object}  map[string]string
// @Router       /auth/login [post]
func (ctrl *AuthController) Login(c *gin.Context) {
	var req model.UserDto
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	user, err := ctrl.userSvc.GetUser(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(strings.TrimSpace(user.Password)), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password"})
		return
	}

	userDto := user.ToDto()
	token, err := auth.GenerateToken(userDto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	ctrl.setAuthCookie(c, token, 1800)
	localCache.UserAuthCache.Delete(req.Email)
	c.JSON(http.StatusOK, userDto)
}

// Logout godoc
// @Summary      User Logout
// @Description  Clears the authentication cookie
// @Tags         Auth
// @Produce      json
// @Success      200    {object}  map[string]string
// @Router       /auth/logout [post]
func (ctrl *AuthController) Logout(c *gin.Context) {
	ctrl.setAuthCookie(c, "", -1)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetMe godoc
// @Summary      Get Current User
// @Description  Retrieves authenticated user details from session
// @Tags         Auth
// @Produce      json
// @Success      200    {object}  model.UserDto
// @Failure      401    {object}  map[string]string
// @Router       /auth/me [get]
func (ctrl *AuthController) GetMe(c *gin.Context) {
	tokenUser, ok := middleware.GetUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	isMobile := false
	cacheKey := tokenUser.Email
	if cacheKey == "" {
		if tokenUser.Mobile == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		cacheKey = fmt.Sprintf("%d", tokenUser.Mobile)
		isMobile = true
	}

	if cached, found := localCache.UserAuthCache.Get(cacheKey); found {
		c.JSON(http.StatusOK, cached.(model.UserDto))
		return
	}

	var user *model.User
	var err error

	if isMobile {
		user, err = ctrl.userSvc.GetUserByMobile(c.Request.Context(), cacheKey)
	} else {
		user, err = ctrl.userSvc.GetUser(c.Request.Context(), cacheKey)
	}

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	dto := user.ToDto()
	localCache.UserAuthCache.Set(cacheKey, dto, cache.DefaultExpiration)
	c.JSON(http.StatusOK, dto)
}

// Signup godoc
// @Summary      User Signup Initiation
// @Description  Caches user data and sends OTP for verification
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        user  body      model.UserDto  true  "Signup Details"
// @Success      200   {object}  model.MessageResponse
// @Failure      409   {object}  model.MessageResponse
// @Router       /auth/signup [post]
func (ctrl *AuthController) Signup(c *gin.Context) {
	var user model.UserDto
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, model.MessageResponse{Message: "Invalid request"})
		return
	}

	localCache.PendingUserCache.Set(user.Email, user, 5*time.Minute)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := ctrl.otpSvc.SendSignUpOtp(ctx, user); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrDuplicateOtp) {
			status = http.StatusConflict
		}
		c.JSON(status, model.MessageResponse{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.MessageResponse{
		OtpSent: true,
		Message: "OTP sent to " + user.Email,
	})
}

// VerifyOtp godoc
// @Summary      Verify OTP and Complete Signup
// @Description  Validates OTP and persists user to database
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      model.VerifyOtpRequest  true  "OTP Verification"
// @Success      201      {object}  model.MessageResponse
// @Failure      400      {object}  model.MessageResponse
// @Router       /auth/verify-otp [post]
func (ctrl *AuthController) VerifyOtp(c *gin.Context) {
	var req model.VerifyOtpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.MessageResponse{Message: "Invalid request"})
		return
	}

	val, found := localCache.PendingUserCache.Get(req.Email)
	if !found {
		c.JSON(http.StatusBadRequest, model.MessageResponse{Message: "Signup session expired"})
		return
	}

	match, err := ctrl.otpSvc.VerifyOtp(req.Email, req.Otp)
	if err != nil || !match {
		c.JSON(http.StatusBadRequest, model.MessageResponse{Message: "Invalid OTP"})
		return
	}

	pendingDto := val.(model.UserDto)
	if _, err := ctrl.userSvc.CreateUser(c.Request.Context(), pendingDto); err != nil {
		c.JSON(http.StatusInternalServerError, model.MessageResponse{Message: "Failed to create user"})
		return
	}

	localCache.PendingUserCache.Delete(req.Email)
	c.JSON(http.StatusCreated, model.MessageResponse{Message: "Signup successful"})
}

// TrueCallerCallBack godoc
// @Summary      Process Truecaller Login Callback
// @Description  Receives the access token from Truecaller, fetches user profile, and warms up the local cache for NSE data.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        data  body      model.TruecallerDto  true  "Truecaller Data"
// @Success      200   {object}  model.Response{success=bool,error=string} "Callback Successful"
// @Failure      400   {object}  model.Response{success=bool,error=string} "Invalid Request"
// @Failure      500   {object}  model.Response{success=bool,error=string} "Internal Server Error"
// @Router       /api/auth/truecaller [post]
func (ctrl *AuthController) TrueCallerCallBack(c *gin.Context) {
	var data model.TruecallerDto
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   "Invalid Request",
		})
		return
	}

	var profile model.TruecallerProfile
	resp, err := ctrl.restyClient.R().
		SetHeader("Authorization", "Bearer "+data.AccessToken).
		SetResult(&profile).
		Get(data.Endpoint)

	if err == nil && resp.IsSuccess() {

		user := model.UserDto{
			Email:    "temp@gmail.com",
			Username: profile.Name.First + "_" + profile.Name.Last,
			Role:     model.RoleUser,
			Theme:    model.ThemeDark,
			Mobile:   profile.PhoneNumbers[0],
		}

		localCache.PendingUserCache.Set(data.RequestId, user, cache.DefaultExpiration)

		c.JSON(http.StatusOK, model.Response{
			Success: true,
			Error:   "Callback Successfull",
		})
		return
	}

	c.JSON(http.StatusInternalServerError, model.Response{
		Success: false,
		Error:   "Invalid Request",
	})
}

// TrueCallerStatus godoc
// @Summary      Check Truecaller Auth Status
// @Description  Polls for the status of a login request. If successful, creates the user, generates a JWT, and sets an auth cookie.
// @Tags         Authentication
// @Produce      json
// @Param        requestId  path      string  true  "Unique Request ID"
// @Success      201        {object}  model.Response{success=bool,message=string,data=model.UserDto} "User created and JWT issued"
// @Failure      404        {object}  model.Response{success=bool,error=string} "Waiting for Truecaller callback"
// @Failure      500        {object}  model.Response{success=bool,message=string} "Failed to create user or generate token"
// @Router       /api/auth/status/{requestId} [get]
func (ctrl *AuthController) TrueCallerStatus(c *gin.Context) {
	reqID := c.Param("requestId")
	if token, ok := localCache.PendingUserCache.Get(reqID); ok {
		userDto := token.(model.UserDto)
		if _, err := ctrl.userSvc.CreateUser(c.Request.Context(), userDto); err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{
				Success: false,
				Message: "Failed to create user"})
			return
		}

		localCache.PendingUserCache.Delete(reqID)
		token, err := auth.GenerateToken(userDto)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{
				Success: false,
				Error:   "Internal server error",
			})
			return
		}

		ctrl.setAuthCookie(c, token, 1800)
		//localCache.UserAuthCache.Delete()
		c.JSON(http.StatusCreated, model.Response{
			Success: true,
			Message: "User created",
			Data:    userDto,
		})
		return
	}

	c.JSON(http.StatusNotFound, model.Response{
		Success: false,
		Error:   "Waiting for truecaller",
	})
}

func (ctrl *AuthController) setAuthCookie(c *gin.Context, token string, maxAge int) {
	if ctrl.isProduction {
		c.SetSameSite(http.SameSiteNoneMode)
	}
	c.SetCookie("auth_token", token, maxAge, "/", "", ctrl.isProduction, true)
}
