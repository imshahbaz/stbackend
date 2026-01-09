package controller

import (
	"context"
	"net/http"

	"backend/config"
	"backend/middleware"
	"backend/model"
	"backend/service"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-webauthn/webauthn/webauthn"
)

var (
	truecaller    = "truecaller_"
	googleProfile = "https://www.googleapis.com/oauth2/v2/userinfo"
)

type AuthController struct {
	userSvc      service.UserService
	cfgManager   *config.ConfigManager
	otpSvc       service.OtpService
	isProduction bool
	oauthSvc     service.OAuthService
	authSvc      service.AuthService
}

func NewAuthController(s service.UserService, cfgManager *config.ConfigManager,
	otpSvc service.OtpService, isProduction bool,
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

func (ctrl *AuthController) RegisterRoutes(api huma.API, w *webauthn.WebAuthn) {
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

	authMw := middleware.HumaAuthMiddleware(api, ctrl.isProduction)

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

	// Public WebAuthn Routes (Login Flow)
	huma.Register(api, huma.Operation{
		OperationID: "webauthn-login-options",
		Method:      http.MethodPost, // Post because we need the email/identifier
		Path:        "/api/auth/webauthn/login/options",
		Summary:     "Get WebAuthn Login Options",
		Tags:        []string{"WebAuthn"},
	}, func(ctx context.Context, input *model.WebAuthnLoginOptionsRequest) (*model.WebAuthnLoginOptionsResponse, error) {
		return ctrl.WebAuthnBeginLogin(ctx, input, w)
	})

	huma.Register(api, huma.Operation{
		OperationID: "webauthn-login-verify",
		Method:      http.MethodPost,
		Path:        "/api/auth/webauthn/login/verify",
		Summary:     "Verify WebAuthn Login",
		Tags:        []string{"WebAuthn"},
	}, func(ctx context.Context, input *model.WebAuthnLoginVerifyRequest) (*model.LoginResponse, error) {
		return ctrl.WebAuthnFinishLogin(ctx, input, w)
	})

	huma.Register(api, huma.Operation{
		OperationID: "webauthn-register-options",
		Method:      http.MethodGet,
		Path:        "/api/auth/webauthn/register/options",
		Middlewares: huma.Middlewares{authMw},
		Summary:     "Get Registration Options",
		Tags:        []string{"WebAuthn"},
	}, func(ctx context.Context, input *struct{}) (*model.WebAuthnOptionsResponse, error) {
		return ctrl.WebAuthnBeginRegistration(ctx, w)
	})

	huma.Register(api, huma.Operation{
		OperationID: "webauthn-register-verify",
		Method:      http.MethodPost,
		Path:        "/api/auth/webauthn/register/verify",
		Middlewares: huma.Middlewares{authMw},
		Summary:     "Verify Registration",
		Tags:        []string{"WebAuthn"},
	}, func(ctx context.Context, input *model.WebAuthnVerifyRequest) (*model.DefaultResponse, error) {
		return ctrl.WebAuthnFinishRegistration(ctx, input, w)
	})

	huma.Register(api, huma.Operation{
		OperationID: "webauthn-toggle",
		Method:      http.MethodPatch,
		Path:        "/api/user/webauthn/toggle",
		Middlewares: huma.Middlewares{authMw},
		Summary:     "Toggle WebAuthn",
		Tags:        []string{"WebAuthn"},
	}, func(ctx context.Context, input *model.WebAuthnToggleRequest) (*model.DefaultResponse, error) {
		return ctrl.WebAuthnToggle(ctx, input)
	})

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

func (ctrl *AuthController) WebAuthnBeginRegistration(ctx context.Context, w *webauthn.WebAuthn) (*model.WebAuthnOptionsResponse, error) {
	user := ctrl.authSvc.GetUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("User not authenticated", nil)
	}

	options, session, err := w.BeginRegistration(user)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to begin registration", err)
	}

	// Store session in Redis/DB using your service
	ctrl.authSvc.SaveWebAuthnSession(user.WebAuthnID(), session)

	resp := &model.WebAuthnOptionsResponse{}
	resp.Body.Success = true
	resp.Body.Data = options
	return resp, nil
}

func (ctrl *AuthController) WebAuthnFinishRegistration(ctx context.Context, input *model.WebAuthnVerifyRequest, w *webauthn.WebAuthn) (*model.DefaultResponse, error) {
	userDto := ctrl.authSvc.GetUserFromContext(ctx)
	if userDto == nil {
		return nil, huma.Error401Unauthorized("User not authenticated", nil)
	}

	session := ctrl.authSvc.GetWebAuthnSession(userDto.WebAuthnID())
	if session == nil {
		return nil, huma.Error400BadRequest("Session expired", nil)
	}

	credential, err := w.CreateCredential(userDto, *session, input.Body.Response)
	if err != nil {
		return nil, huma.Error400BadRequest("Verification failed", err)
	}

	// Persist credential to DB via user service
	userDto.AddCredential(*credential)
	userDto.WebAuthnEnabled = true

	user, err := userDto.ToEntity()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to process user data", err)
	}

	ctrl.userSvc.PatchUserData(ctx, user.UserID, *user)

	return &model.DefaultResponse{Body: model.Response{Success: true, Message: "Linked"}}, nil
}

func (ctrl *AuthController) WebAuthnBeginLogin(ctx context.Context, input *model.WebAuthnLoginOptionsRequest, w *webauthn.WebAuthn) (*model.WebAuthnLoginOptionsResponse, error) {
	var mobile int64
	var email string

	// Parse identifier into either mobile (int64) or email (string)
	if m, parseErr := strconv.ParseInt(input.Body.Identifier, 10, 64); parseErr == nil && len(input.Body.Identifier) >= 10 {
		mobile = m
	} else {
		email = input.Body.Identifier
	}

	user, err := ctrl.userSvc.FindUser(ctx, mobile, email, 0)
	if err != nil {
		return nil, huma.Error404NotFound("User not found", err)
	}

	options, session, err := w.BeginLogin(user)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to begin login", err)
	}

	ctrl.authSvc.SaveWebAuthnSession(user.WebAuthnID(), session)

	resp := &model.WebAuthnLoginOptionsResponse{}
	resp.Body.Success = true
	resp.Body.Data = options
	return resp, nil
}

func (ctrl *AuthController) WebAuthnFinishLogin(ctx context.Context, input *model.WebAuthnLoginVerifyRequest, w *webauthn.WebAuthn) (*model.LoginResponse, error) {

	uidStr := string(input.Body.Response.Response.UserHandle)
	userId, _ := strconv.ParseInt(uidStr, 10, 64)

	session := ctrl.authSvc.GetWebAuthnSession([]byte(uidStr))
	if session == nil {
		return nil, huma.Error400BadRequest("Login session expired", nil)
	}

	user, err := ctrl.userSvc.FindUser(ctx, 0, "", userId)
	if err != nil {
		return nil, huma.Error404NotFound("User not found", nil)
	}

	_, err = w.ValidateLogin(user, *session, input.Body.Response)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication failed", err)
	}

	// Reuse user existing login logic to create JWT/Cookies
	return ctrl.authSvc.CreateAuthSession(ctx, user)
}

func (ctrl *AuthController) WebAuthnToggle(ctx context.Context, input *model.WebAuthnToggleRequest) (*model.DefaultResponse, error) {
	userDto := ctrl.authSvc.GetUserFromContext(ctx)
	if userDto == nil {
		return nil, huma.Error401Unauthorized("User not authenticated", nil)
	}

	user, err := userDto.ToEntity()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to process user data", err)
	}

	user.WebAuthnEnabled = input.Body.Enabled
	ctrl.userSvc.PatchUserData(ctx, user.UserID, *user)

	return &model.DefaultResponse{Body: model.Response{Success: true, Message: "Updated"}}, nil
}
