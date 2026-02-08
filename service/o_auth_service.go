package service

import (
	"backend/auth"
	"backend/cache"
	"backend/config"
	"backend/customerrors"
	"backend/database"
	"backend/model"
	"backend/util"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

type OAuthService interface {
	ValidateToken(ctx context.Context, input *model.AuthInput) (*model.GoogleAuthResponse, error)
	TrueCallerCallBack(ctx context.Context, input *model.Request) (*model.ResponseWrapper, error)
	TrueCallerStatus(ctx context.Context, requestId string) (*model.DetailedResponseWrapper, error)
	GoogleAuthCallback(ctx context.Context, input *model.AuthInput) (*model.GoogleAuthResponse, error)
}

type OAuthServiceImpl struct {
	userSvc      UserService
	cfgManager   *config.ConfigManager
	restyClient  *resty.Client
	googleConfig *oauth2.Config
	isProduction IsProduction
}

func NewOAuthService(userSvc UserService, cfgManager *config.ConfigManager, isProduction IsProduction, googleConfig *oauth2.Config) OAuthService {
	return &OAuthServiceImpl{
		userSvc:      userSvc,
		cfgManager:   cfgManager,
		restyClient:  resty.New().SetTimeout(10 * time.Second),
		googleConfig: googleConfig,
		isProduction: isProduction,
	}
}

func (svc *OAuthServiceImpl) ValidateToken(ctx context.Context, input *model.AuthInput) (*model.GoogleAuthResponse, error) {
	id := uuid.New().String()
	signUuid := util.SignState(id, svc.cfgManager.GetConfig().GoogleAuth.EncryptionKey)
	go func() {
		detachedCtx := context.Background()
		gUser, err := util.ValidateGoogleIDToken(detachedCtx, input.Code, svc.cfgManager.GetConfig().GoogleAuth.ClientID)
		if err != nil {
			log.Warn().Err(err).Msg("Invalid Google token")
			return
		}

		user, err := svc.findOrCreateGoogleUser(detachedCtx, *gUser)
		if err != nil {
			log.Error().Err(err).Msg("Failed to find or create google user")
			return
		}

		svc.patchUserMetadata(detachedCtx, user, gUser.Picture, "")
		cache.GoSet("auth_"+id, user.ToDto(), 2*time.Minute)
	}()

	return &model.GoogleAuthResponse{
		Status: http.StatusOK,
		Body:   model.Response{Data: signUuid},
	}, nil
}

func (svc *OAuthServiceImpl) findOrCreateGoogleUser(ctx context.Context, gUser model.GoogleUser) (*model.User, error) {
	user, err := svc.userSvc.FindUser(ctx, 0, gUser.Email, 0)
	if err != nil && !errors.Is(err, customerrors.ErrUserNotFound) {
		return nil, err
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
		return svc.userSvc.CreateUser(ctx, dto)
	}
	return user, nil
}

func (svc *OAuthServiceImpl) patchUserMetadata(ctx context.Context, user *model.User, profile, mobile string) {
	updated := false
	patch := model.User{}

	if profile != "" && (user.Profile == "" || user.Profile != profile) {
		patch.Profile = profile
		user.Profile = profile
		updated = true
	}

	m, _ := strconv.ParseInt(mobile, 10, 64)
	if m > 0 && (user.Mobile == 0 || user.Mobile != m) {
		patch.Mobile = m
		user.Mobile = m
		updated = true
	}

	if updated {
		if err := svc.userSvc.PatchUserData(ctx, user.UserID, patch); err != nil {
			log.Warn().Err(err).Msgf("Failed to patch user metadata for ID: %d", user.UserID)
		}
	}
}

func (svc *OAuthServiceImpl) TrueCallerCallBack(ctx context.Context, input *model.Request) (*model.ResponseWrapper, error) {
	var body model.TruecallerDto
	if err := mapstructure.Decode(input.Body, &body); err != nil {
		return nil, huma.Error400BadRequest("Invalid Request")
	}

	switch body.Status {
	case "user_rejected":
		return nil, huma.Error400BadRequest("User rejected the Truecaller authentication")
	case "flow_invoked":
		log.Info().Msgf("Handshake received for Nonce: %s", body.RequestId)
		return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Flow invocation success"}}, nil
	}

	detachedCtx := context.WithoutCancel(ctx)
	var profile model.TruecallerProfile
	resp, err := svc.restyClient.R().
		SetHeader("Authorization", "Bearer "+body.AccessToken).
		SetHeader("Cache-Control", "no-cache").
		SetResult(&profile).
		Get(body.Endpoint)

	if err != nil || !resp.IsSuccess() {
		return nil, huma.Error500InternalServerError("Failed to fetch Truecaller profile")
	}

	user, err := svc.userSvc.FindUser(detachedCtx, profile.PhoneNumbers[0], profile.OnlineIdentities.Email, 0)
	if err != nil && !errors.Is(err, customerrors.ErrUserNotFound) {
		return nil, huma.Error500InternalServerError("Internal server error while finding user")
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

		if user, err = svc.userSvc.CreateUser(detachedCtx, dto); err != nil {
			return nil, huma.Error500InternalServerError("Failed to create user from Truecaller profile")
		}
	} else {
		svc.patchUserMetadata(detachedCtx, user, "", strconv.FormatInt(profile.PhoneNumbers[0], 10))
	}

	cache.SetUserCache(body.RequestId, user.ToDto(), model.Truecaller)
	return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Callback Successfull"}}, nil
}

func (svc *OAuthServiceImpl) TrueCallerStatus(ctx context.Context, requestId string) (*model.DetailedResponseWrapper, error) {
	var userDto model.UserDto
	if ok, _ := cache.GetUserCache(requestId, &userDto, model.Truecaller); ok {
		tokenStr, err := auth.GenerateToken(userDto)
		if err != nil {
			log.Info().Msgf("Error while generating token %v", err.Error())
			return nil, huma.Error500InternalServerError("Internal server error")
		}

		cookie := svc.createAuthCookie(tokenStr, 1800)
		cache.GoSet("auth_"+strconv.FormatInt(userDto.UserID, 10), userDto, time.Hour)

		return &model.DetailedResponseWrapper{
			SetCookie: cookie,
			Body: model.Response{
				Success: true,
				Message: "User authenticated",
				Data:    userDto,
			},
		}, nil
	}

	return nil, huma.Error404NotFound("Waiting for Truecaller")
}

func (svc *OAuthServiceImpl) createAuthCookie(token string, maxAge int) string {
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    token,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   bool(svc.isProduction),
		HttpOnly: true,
	}
	if svc.isProduction {
		cookie.SameSite = http.SameSiteNoneMode
	}
	return cookie.String()
}

func (svc *OAuthServiceImpl) GoogleAuthCallback(ctx context.Context, input *model.AuthInput) (*model.GoogleAuthResponse, error) {
	var targetURL string
	isIPhoneRedirect := false

	if strings.HasPrefix(input.State, "redirect|") {
		parts := strings.Split(input.State, "|")
		if len(parts) == 2 {
			potentialTarget := parts[1]
			for _, allowed := range svc.cfgManager.GetConfig().FrontendUrls {
				if strings.HasPrefix(potentialTarget, allowed) {
					isIPhoneRedirect = true
					targetURL = potentialTarget
					break
				}
			}

			if !isIPhoneRedirect {
				return nil, huma.Error400BadRequest("Unauthorized redirect origin")
			}

			id := uuid.New().String()
			signUuid := util.SignState(id, svc.cfgManager.GetConfig().GoogleAuth.EncryptionKey)
			targetURL = targetURL + "/google/callback?code=" + signUuid + "&state=standard"
			go func() {
				svc.googleCallbackProcessing(context.Background(), input.Code, id)
			}()

			return &model.GoogleAuthResponse{
				Status:   http.StatusTemporaryRedirect,
				Location: targetURL,
			}, nil
		}
	}

	if input.State == "standard" {
		id, ok := util.ExtractAndVerify(input.Code, svc.cfgManager.GetConfig().GoogleAuth.EncryptionKey)
		if !ok {
			return nil, huma.Error400BadRequest("Invalid or tampered session state")
		}

		var userDto model.UserDto
		key := "auth_" + id
		if ok, _ := database.RedisHelper.GetAsStruct(key, &userDto); !ok {
			return nil, huma.Error404NotFound("Request still under process or expired")
		}

		tokenStr, err := auth.GenerateToken(userDto)
		if err != nil {
			log.Info().Msgf("Error while generating token %v", err.Error())
			return nil, huma.Error500InternalServerError("Internal server error")
		}

		cookie := svc.createAuthCookie(tokenStr, 1800)
		cache.GoSet("auth_"+strconv.FormatInt(userDto.UserID, 10), userDto, time.Hour)
		cache.GoDelete(key)
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

	return nil, huma.Error401Unauthorized("Invalid state")
}

func (svc *OAuthServiceImpl) googleCallbackProcessing(ctx context.Context, code, uuid string) {
	conf := *svc.googleConfig
	token, err := conf.Exchange(ctx, code)
	if err != nil {
		log.Err(err).Msgf("Exchange failed with google %v %v", uuid, err)
		return
	}

	client := conf.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Err(err).Msgf("Exchange failed with google %v %v", uuid, err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var gUser model.GoogleUser
	if err := json.Unmarshal(bodyBytes, &gUser); err != nil {
		log.Err(err).Msgf("JSON decode failed %v %v", uuid, err)
		return
	}

	user, err := svc.findOrCreateGoogleUser(ctx, gUser)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to find or create google user %v", uuid)
		return
	}

	svc.patchUserMetadata(ctx, user, gUser.Picture, "")
	cache.GoSet("auth_"+uuid, user.ToDto(), 2*time.Minute)
}
