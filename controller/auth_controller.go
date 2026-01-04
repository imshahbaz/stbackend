package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/auth"
	localCache "backend/cache"
	"backend/config"
	"backend/customerrors"
	"backend/database"
	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-resty/resty/v2"
	"github.com/jinzhu/copier"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
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
	restyClient  *resty.Client
	googleConfig *oauth2.Config
}

func NewAuthController(s service.UserService, cfgManager *config.ConfigManager,
	otpSvc service.OtpService, isProduction bool, googleConfig *oauth2.Config) *AuthController {
	return &AuthController{
		userSvc:      s,
		cfgManager:   cfgManager,
		otpSvc:       otpSvc,
		isProduction: isProduction,
		restyClient:  resty.New().SetTimeout(10 * time.Second),
		googleConfig: googleConfig,
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
	database.RedisHelper.Delete("auth_" + strconv.FormatInt(userDto.UserID, 10))

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
	localCache.SetUserCache(user.Email, userDto, model.Signup)

	ctxt, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := ctrl.otpSvc.SendOtp(ctxt, user.Email, model.OTPRegister); err != nil {
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
	var pendingDto model.UserDto
	ok, err := localCache.GetUserCache(req.Email, &pendingDto, model.Signup)
	if err != nil || !ok {
		return nil, huma.Error400BadRequest("Signup session expired")
	}

	match, err := ctrl.otpSvc.VerifyOtp(req.Email, req.Otp, model.OTPRegister)
	if err != nil || !match {
		return nil, huma.Error400BadRequest("Invalid OTP")
	}

	if _, err := ctrl.userSvc.CreateUser(ctx, pendingDto); err != nil {
		return nil, huma.Error500InternalServerError("Failed to create user")
	}

	localCache.DeleteUserCache(req.Email, model.Signup)
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

	cacheKey := "auth_" + strconv.FormatInt(tokenUser.UserID, 10)
	var dto model.UserDto
	if ok, _ := database.RedisHelper.GetAsStruct(cacheKey, &dto); ok {
		return &model.LoginResponse{Body: model.Response{
			Success: true,
			Message: "User details fetched",
			Data:    dto,
		}}, nil
	}

	user, err := ctrl.userSvc.FindUser(ctx, tokenUser.Mobile, tokenUser.Email, tokenUser.UserID)
	if err != nil {
		return nil, huma.Error401Unauthorized("User not found")
	}

	dto = user.ToDto()
	database.RedisHelper.Set(cacheKey, dto, time.Hour)
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
		log.Info().Msgf("Handshake received for Nonce: %s", body.RequestId)
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

		if profile.PhoneNumbers[0] > 0 && (user.Mobile == 0 || user.Mobile != profile.PhoneNumbers[0]) {
			if err := ctrl.userSvc.PatchUserData(detachedCtx, user.UserID, model.User{
				Mobile: profile.PhoneNumbers[0],
			}); err != nil {
				log.Info().Msgf("Unable to update mobile number userId : %v", user.UserID)
			} else {
				user.Mobile = profile.PhoneNumbers[0]
			}
		}

		localCache.SetUserCache(body.RequestId, user.ToDto(), model.Truecaller)

		return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Callback Successfull"}}, nil
	}

	return nil, huma.Error500InternalServerError("Invalid Request")
}

func (ctrl *AuthController) TrueCallerStatus(ctx context.Context, input *model.TrueCallerStatusInput) (*model.DetailedResponseWrapper, error) {
	reqID := input.RequestId
	var userDto model.UserDto
	if ok, _ := localCache.GetUserCache(reqID, &userDto, model.Truecaller); ok {
		tokenStr, err := auth.GenerateToken(userDto)
		if err != nil {
			log.Info().Msgf("Error while generating token %v", err.Error())
			return nil, huma.Error500InternalServerError("Internal server error")
		}

		cookie := ctrl.createAuthCookie(tokenStr, 1800)
		database.RedisHelper.Set("auth_"+strconv.FormatInt(userDto.UserID, 10), userDto, time.Hour)

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

func (ctrl *AuthController) googleAuthCallback(ctx context.Context, input *model.AuthInput) (*model.GoogleAuthResponse, error) {
	conf := *ctrl.googleConfig

	var targetURL string
	isIPhoneRedirect := false

	if strings.HasPrefix(input.State, "redirect|") {
		parts := strings.Split(input.State, "|")
		if len(parts) == 2 {
			potentialTarget := parts[1]
			for _, allowed := range ctrl.cfgManager.GetConfig().FrontendUrls {
				if strings.HasPrefix(potentialTarget, allowed) {
					isIPhoneRedirect = true
					targetURL = potentialTarget
					break
				}
			}

			if !isIPhoneRedirect {
				return nil, huma.Error400BadRequest("Unauthorized redirect origin")
			}
		}
	}

	if isIPhoneRedirect {
		conf.RedirectURL = ctrl.cfgManager.GetConfig().GoogleAuth.CallbackUrl
	} else {
		conf.RedirectURL = "postmessage"
	}

	detachedCtx := context.WithoutCancel(ctx)
	token, err := conf.Exchange(detachedCtx, input.Code)
	if err != nil {
		return nil, huma.Error401Unauthorized("Exchange failed", err)
	}

	client := conf.Client(detachedCtx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, huma.Error401Unauthorized("Exchange failed", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var gUser model.GoogleUser
	if err := json.Unmarshal(bodyBytes, &gUser); err != nil {
		return nil, huma.Error500InternalServerError("JSON decode failed", err)
	}

	user, err := ctrl.userSvc.FindUser(detachedCtx, 0, gUser.Email, 0)
	if err != nil && !errors.Is(err, customerrors.ErrUserNotFound) {
		return nil, huma.Error400BadRequest("Invalid Request")
	}

	if user == nil {
		dto := model.UserDto{
			Email:    gUser.Email,
			Username: gUser.GivenName + "_" + gUser.FamilyName,
			Role:     model.RoleUser,
			Theme:    model.ThemeDark,
			Name:     gUser.Name,
			Profile:  gUser.Picture,
		}

		newUser, err := ctrl.userSvc.CreateUser(detachedCtx, dto)
		if err != nil {
			return nil, huma.Error500InternalServerError("Invalid Request")
		}

		user = newUser
	}

	if gUser.Picture != "" && (user.Profile == "" || gUser.Picture != user.Profile) {
		if err := ctrl.userSvc.PatchUserData(detachedCtx, user.UserID, model.User{
			Profile: gUser.Picture,
		}); err != nil {
			log.Info().Msgf("Unable to update profile picture userId : %v", user.UserID)
		} else {
			user.Profile = gUser.Picture
		}
	}

	userDto := user.ToDto()
	tokenStr, err := auth.GenerateToken(userDto)
	if err != nil {
		log.Info().Msgf("Error while generating token %v", err.Error())
		return nil, huma.Error500InternalServerError("Internal server error")
	}

	cookie := ctrl.createAuthCookie(tokenStr, 1800)
	database.RedisHelper.Set("auth_"+strconv.FormatInt(userDto.UserID, 10), userDto, time.Hour)

	if isIPhoneRedirect {
		return &model.GoogleAuthResponse{
			Status:    http.StatusFound,
			SetCookie: cookie,
			Location:  targetURL,
		}, nil
	}

	return &model.GoogleAuthResponse{
		Status:    http.StatusOK,
		SetCookie: cookie,
		Body: model.Response{
			Success: true,
			Message: "User created",
			Data:    userDto,
		},
	}, nil
}
