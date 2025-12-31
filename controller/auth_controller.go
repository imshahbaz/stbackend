package controller

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/auth"
	localCache "backend/cache"
	"backend/config"
	"backend/customerrors"
	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-resty/resty/v2"
	"github.com/jinzhu/copier"
	"github.com/mitchellh/mapstructure"
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

	// Protected routes
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

	// TrueCaller
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
}

func (ctrl *AuthController) Login(ctx context.Context, input *model.LoginRequest) (*model.LoginResponse, error) {
	req := input.Body
	user, err := ctrl.userSvc.FindUser(ctx, 0, req.Email, 0)
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

	cookie := ctrl.createAuthCookie(token, 1800)
	localCache.UserAuthCache.Delete(strconv.FormatInt(userDto.UserID, 10))

	return &model.LoginResponse{
		SetCookie: cookie,
		Body: model.Response{
			Success: true,
			Message: "Login successful",
			Data:    userDto,
		},
	}, nil
}

func (ctrl *AuthController) Signup(ctx context.Context, input *model.SignupRequest) (*model.MessageResponseWrapper, error) {
	user := input.Body
	var userDto model.UserDto
	copier.Copy(&userDto, &user)
	localCache.PendingUserCache.Set(user.Email, userDto, 5*time.Minute)

	ctxt, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := ctrl.otpSvc.SendSignUpOtp(ctxt, user); err != nil {
		if errors.Is(err, service.ErrDuplicateOtp) {
			return nil, huma.Error409Conflict(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &model.MessageResponseWrapper{
		Body: model.Response{
			Success: true,
			Message: "OTP sent to " + user.Email,
			Data:    model.MessageResponse{OtpSent: true, Message: "OTP sent to " + user.Email},
		},
	}, nil
}

func (ctrl *AuthController) VerifyOtp(ctx context.Context, input *model.VerifyOtpInput) (*model.MessageResponseWrapper, error) {
	req := input.Body
	val, found := localCache.PendingUserCache.Get(req.Email)
	if !found {
		return nil, huma.Error400BadRequest("Signup session expired")
	}

	match, err := ctrl.otpSvc.VerifyOtp(req.Email, req.Otp)
	if err != nil || !match {
		return nil, huma.Error400BadRequest("Invalid OTP")
	}

	pendingDto := val.(model.UserDto)
	if _, err := ctrl.userSvc.CreateUser(ctx, pendingDto); err != nil {
		return nil, huma.Error500InternalServerError("Failed to create user")
	}

	localCache.PendingUserCache.Delete(req.Email)
	return &model.MessageResponseWrapper{Body: model.Response{Success: true, Message: "Signup successful"}}, nil
}

func (ctrl *AuthController) Logout(ctx context.Context, input *struct{}) (*model.LogoutResponse, error) {
	cookie := ctrl.createAuthCookie("", -1)
	return &model.LogoutResponse{
		SetCookie: cookie,
		Body:      model.Response{Success: true, Message: "Logged out successfully"},
	}, nil
}

func (ctrl *AuthController) GetMe(ctx context.Context, input *struct{}) (*model.LoginResponse, error) {
	val := ctx.Value("user")
	if val == nil {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}
	tokenUser := val.(model.UserDto)

	cacheKey := strconv.FormatInt(tokenUser.UserID, 10)
	if cached, found := localCache.UserAuthCache.Get(cacheKey); found {
		return &model.LoginResponse{Body: model.Response{
			Success: true,
			Message: "User details fetched",
			Data:    cached.(model.UserDto),
		}}, nil
	}

	user, err := ctrl.userSvc.FindUser(ctx, tokenUser.Mobile, tokenUser.Email, tokenUser.UserID)
	if err != nil {
		return nil, huma.Error401Unauthorized("User not found")
	}

	dto := user.ToDto()
	localCache.UserAuthCache.Set(cacheKey, dto, cache.DefaultExpiration)
	return &model.LoginResponse{Body: model.Response{
		Success: true,
		Message: "User details fetched",
		Data:    dto,
	}}, nil
}

func (ctrl *AuthController) TrueCallerCallBack(ctx context.Context, input *model.Request) (*model.ResponseWrapper, error) {

	var body model.TruecallerDto

	if err := mapstructure.Decode(input.Body, &body); err != nil {
		return nil, huma.Error400BadRequest("Invalid Request")
	}

	if body.Status == "user_rejected" {
		return nil, huma.Error400BadRequest("User rejected the Truecaller authentication")
	}

	if body.Status == "flow_invoked" {
		log.Printf("Handshake received for Nonce: %s", body.RequestId)
		return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Flow invocation success"}}, nil
	}

	detachedCtx := context.WithoutCancel(ctx)

	var profile model.TruecallerProfile
	resp, err := ctrl.restyClient.R().
		SetHeader("Authorization", "Bearer "+body.AccessToken).
		SetHeader("Cache-Control", "no-cache").
		SetResult(&profile).
		Get(body.Endpoint)

	if err == nil && resp.IsSuccess() {
		user, err := ctrl.userSvc.FindUser(detachedCtx, profile.PhoneNumbers[0], profile.OnlineIdentities.Email, 0)
		if err != nil && !errors.Is(err, customerrors.ErrUserNotFound) {
			return nil, huma.Error400BadRequest("Invalid Request")
		}

		if user == nil {
			dto := model.UserDto{
				Email:    profile.OnlineIdentities.Email,
				Username: profile.Name.First + "_" + profile.Name.Last,
				Role:     model.RoleUser,
				Theme:    model.ThemeDark,
				Mobile:   profile.PhoneNumbers[0],
				Name:     strings.TrimSpace(profile.Name.First + " " + profile.Name.Last),
			}

			newUser, err := ctrl.userSvc.CreateUser(detachedCtx, dto)
			if err != nil {
				return nil, huma.Error500InternalServerError("Invalid Request")
			}

			user = newUser
		}

		localCache.PendingUserCache.Set(body.RequestId, user.ToDto(), cache.DefaultExpiration)

		return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Callback Successfull"}}, nil
	}

	return nil, huma.Error500InternalServerError("Invalid Request")
}

func (ctrl *AuthController) TrueCallerStatus(ctx context.Context, input *model.TrueCallerStatusInput) (*model.DetailedResponseWrapper, error) {
	reqID := input.RequestId
	if token, ok := localCache.PendingUserCache.Get(reqID); ok {
		userDto := token.(model.UserDto)
		localCache.PendingUserCache.Delete(reqID)
		tokenStr, err := auth.GenerateToken(userDto)
		if err != nil {
			log.Printf("Error while generating token %v", err.Error())
			return nil, huma.Error500InternalServerError("Internal server error")
		}

		cookie := ctrl.createAuthCookie(tokenStr, 1800)
		localCache.UserAuthCache.Set(strconv.FormatInt(userDto.UserID, 10), userDto, cache.DefaultExpiration)

		return &model.DetailedResponseWrapper{
			SetCookie: cookie,
			Body: model.Response{
				Success: true,
				Message: "User created",
				Data:    userDto,
			},
		}, nil
	}

	return nil, huma.Error404NotFound("Waiting for truecaller")
}

func (ctrl *AuthController) createAuthCookie(token string, maxAge int) string {
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    token,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   ctrl.isProduction,
		HttpOnly: true,
	}
	if ctrl.isProduction {
		cookie.SameSite = http.SameSiteNoneMode
	}
	return cookie.String()
}
