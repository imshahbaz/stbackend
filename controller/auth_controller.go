package controller

import (
	"context"
	"net/http"

	"backend/config"
	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

var (
	truecaller    = "truecaller_"
	googleProfile = "https://www.googleapis.com/oauth2/v2/userinfo"
)

type AuthController struct {
	userSvc      service.UserService
	cfgManager   *config.ConfigManager
	otpSvc       service.OtpService
	isProduction service.IsProduction
	oauthSvc     service.OAuthService
	authSvc      service.AuthService
}

func NewAuthController(s service.UserService, cfgManager *config.ConfigManager,
	otpSvc service.OtpService, isProduction service.IsProduction,
	oauthSvc service.OAuthService, authSvc service.AuthService) *AuthController {
	return &AuthController{
		userSvc:      s,
		cfgManager:   cfgManager,
		otpSvc:       otpSvc,
		isProduction: isProduction,
		oauthSvc:     oauthSvc,
		authSvc:      authSvc,
	}
}

func (ctrl *AuthController) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/api/auth/login",
		Summary:     "User Login",
		Description: "Authenticates user via HttpOnly cookie and JWT",
		Tags:        []string{"Auth"},
	}, ctrl.Login)

	huma.Register(api, huma.Operation{
		OperationID: "signup",
		Method:      http.MethodPost,
		Path:        "/api/auth/signup",
		Summary:     "User Signup Initiation",
		Tags:        []string{"Auth"},
	}, ctrl.Signup)

	huma.Register(api, huma.Operation{
		OperationID: "verify-otp",
		Method:      http.MethodPost,
		Path:        "/api/auth/verify-otp",
		Summary:     "Verify OTP",
		Tags:        []string{"Auth"},
	}, ctrl.VerifyOtp)

	authMw := middleware.HumaAuthMiddleware(api, bool(ctrl.isProduction))

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/api/auth/logout",
		Summary:     "User Logout",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Auth"},
	}, ctrl.Logout)

	huma.Register(api, huma.Operation{
		OperationID: "get-me",
		Method:      http.MethodGet,
		Path:        "/api/auth/me",
		Summary:     "Get Current User",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Auth"},
	}, ctrl.GetMe)

	huma.Register(api, huma.Operation{
		OperationID: "truecaller-callback",
		Method:      http.MethodPost,
		Path:        "/api/auth/truecaller",
		Summary:     "Process Truecaller Login Callback",
		Tags:        []string{"Authentication"},
	}, ctrl.TrueCallerCallBack)

	huma.Register(api, huma.Operation{
		OperationID: "truecaller-status",
		Method:      http.MethodGet,
		Path:        "/api/auth/truecaller/status/{requestId}",
		Summary:     "Check Truecaller Auth Status",
		Tags:        []string{"Authentication"},
	}, ctrl.TrueCallerStatus)

	huma.Register(api, huma.Operation{
		OperationID: "google-callback",
		Method:      http.MethodGet,
		Path:        "/api/auth/google/callback",
		Summary:     "Process Google OAuth Callback",
		Tags:        []string{"Authentication"},
	}, ctrl.googleAuthCallback)

	huma.Register(api, huma.Operation{
		OperationID: "google-token-validation",
		Method:      http.MethodPost,
		Path:        "/api/auth/google/token",
		Summary:     "Process Google Token",
		Tags:        []string{"Authentication"},
	}, ctrl.validateGoogleToken)
}

func (ctrl *AuthController) Login(ctx context.Context, input *model.LoginRequest) (*model.LoginResponse, error) {
	return ctrl.authSvc.Login(ctx, input.Body)
}

func (ctrl *AuthController) Signup(ctx context.Context, input *model.SignupRequest) (*model.MessageResponseWrapper, error) {
	return ctrl.authSvc.Signup(ctx, input.Body)
}

func (ctrl *AuthController) VerifyOtp(ctx context.Context, input *model.VerifyOtpInput) (*model.MessageResponseWrapper, error) {
	return ctrl.authSvc.VerifyOtp(ctx, input.Body)
}

func (ctrl *AuthController) Logout(ctx context.Context, input *struct{}) (*model.LogoutResponse, error) {
	return ctrl.authSvc.Logout(), nil
}

func (ctrl *AuthController) GetMe(ctx context.Context, input *struct{}) (*model.LoginResponse, error) {
	return ctrl.authSvc.GetMe(ctx)
}

func (ctrl *AuthController) TrueCallerCallBack(ctx context.Context, input *model.Request) (*model.ResponseWrapper, error) {
	return ctrl.oauthSvc.TrueCallerCallBack(ctx, input)
}

func (ctrl *AuthController) TrueCallerStatus(ctx context.Context, input *model.TrueCallerStatusInput) (*model.DetailedResponseWrapper, error) {
	return ctrl.oauthSvc.TrueCallerStatus(ctx, input.RequestId)
}

func (ctrl *AuthController) googleAuthCallback(ctx context.Context, input *model.AuthInput) (*model.GoogleAuthResponse, error) {
	return ctrl.oauthSvc.GoogleAuthCallback(ctx, input)
}

func (ctrl *AuthController) validateGoogleToken(ctx context.Context, input *model.AuthInput) (*model.GoogleAuthResponse, error) {
	return ctrl.oauthSvc.ValidateToken(ctx, input)
}
