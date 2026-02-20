package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"backend/auth"
	"backend/cache"
	"backend/database"
	"backend/model"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jinzhu/copier"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Login(ctx context.Context, req model.LoginDto) (*model.LoginResponse, error)
	Signup(ctx context.Context, req model.SignupDto) (*model.MessageResponseWrapper, error)
	VerifyOtp(ctx context.Context, req model.VerifyOtpRequest) (*model.MessageResponseWrapper, error)
	Logout() *model.LogoutResponse
	GetMe(ctx context.Context) (*model.LoginResponse, error)
}

type AuthServiceImpl struct {
	userSvc      UserService
	otpSvc       OtpService
	isProduction IsProduction
}

func NewAuthService(userSvc UserService, otpSvc OtpService, isProduction IsProduction) AuthService {
	return &AuthServiceImpl{
		userSvc:      userSvc,
		otpSvc:       otpSvc,
		isProduction: isProduction,
	}
}

func (s *AuthServiceImpl) Login(ctx context.Context, req model.LoginDto) (*model.LoginResponse, error) {
	user, err := s.userSvc.FindUser(ctx, 0, req.Email, 0)
	if err != nil {
		return nil, huma.Error401Unauthorized("Invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(strings.TrimSpace(user.Password)), []byte(req.Password)); err != nil {
		return nil, huma.Error401Unauthorized("Invalid email or password")
	}

	userDto := user.ToDto()
	token, err := auth.GenerateToken(userDto)
	if err != nil {
		return nil, huma.Error500InternalServerError("Internal server error")
	}

	cache.GoDelete(s.getAuthCacheKey(userDto.UserID))
	return s.wrapLoginResponse(userDto, token, "Login successful"), nil
}

func (s *AuthServiceImpl) Signup(ctx context.Context, req model.SignupDto) (*model.MessageResponseWrapper, error) {
	var userDto model.UserDto
	copier.Copy(&userDto, &req)
	cache.SetUserCache(req.Email, userDto, model.Signup)

	ctxt, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.otpSvc.SendOtp(ctxt, req.Email, model.OTPRegister); err != nil {
		if errors.Is(err, ErrDuplicateOtp) {
			return nil, huma.Error409Conflict(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &model.MessageResponseWrapper{
		Body: model.Response{
			Success: true,
			Message: "OTP sent to " + req.Email,
			Data:    model.MessageResponse{OtpSent: true, Message: "OTP sent to " + req.Email},
		},
	}, nil
}

func (s *AuthServiceImpl) VerifyOtp(ctx context.Context, req model.VerifyOtpRequest) (*model.MessageResponseWrapper, error) {
	var pendingDto model.UserDto
	ok, err := cache.GetUserCache(req.Email, &pendingDto, model.Signup)
	if err != nil || !ok {
		return nil, huma.Error400BadRequest("Signup session expired")
	}

	match, err := s.otpSvc.VerifyOtp(req.Email, req.Otp, model.OTPRegister)
	if err != nil || !match {
		return nil, huma.Error400BadRequest("Invalid OTP")
	}

	if _, err := s.userSvc.CreateUser(ctx, pendingDto); err != nil {
		return nil, huma.Error500InternalServerError("Failed to create user")
	}

	cache.DeleteUserCache(req.Email, model.Signup)
	return &model.MessageResponseWrapper{Body: model.Response{Success: true, Message: "Signup successful"}}, nil
}

func (s *AuthServiceImpl) Logout() *model.LogoutResponse {
	cookie := s.createAuthCookie("", -1)
	return &model.LogoutResponse{
		SetCookie: cookie,
		Body:      model.Response{Success: true, Message: "Logged out successfully"},
	}
}

func (s *AuthServiceImpl) GetMe(ctx context.Context) (*model.LoginResponse, error) {
	val := ctx.Value("user")
	if val == nil {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}
	tokenUser := val.(model.UserDto)

	cacheKey := s.getAuthCacheKey(tokenUser.UserID)
	var dto model.UserDto
	if ok, _ := database.RedisHelper.GetAsStruct(cacheKey, &dto); ok {
		return s.wrapLoginResponse(dto, "", "User details fetched"), nil
	}

	user, err := s.userSvc.FindUser(ctx, tokenUser.Mobile, tokenUser.Email, tokenUser.UserID)
	if err != nil {
		return nil, huma.Error401Unauthorized("User not found")
	}

	dto = user.ToDto()
	cache.GoSet(cacheKey, dto, time.Hour)
	return s.wrapLoginResponse(dto, "", "User details fetched"), nil
}

func (s *AuthServiceImpl) createAuthCookie(token string, maxAge int) string {
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    token,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   bool(s.isProduction),
		HttpOnly: true,
	}
	if s.isProduction {
		cookie.SameSite = http.SameSiteNoneMode
	}
	return cookie.String()
}

func (s *AuthServiceImpl) getAuthCacheKey(userID int64) string {
	return "auth_" + strconv.FormatInt(userID, 10)
}

func (s *AuthServiceImpl) wrapLoginResponse(user model.UserDto, token string, message string) *model.LoginResponse {
	resp := &model.LoginResponse{
		Body: model.Response{
			Success: true,
			Message: message,
			Data:    user,
		},
	}
	if token != "" {
		resp.SetCookie = s.createAuthCookie(token, 86400)
	}
	return resp
}
