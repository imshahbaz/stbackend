package controller

import (
	"backend/cache"
	"backend/database"
	"backend/middleware"
	"backend/model"
	"backend/service"
	"backend/util"
	"context"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

type ZerodhaController struct {
	zerodhaSvc   service.ZerodhaService
	isProduction service.IsProduction
	userSvc      service.UserService
}

func NewZerodhaController(zerodhaSvc service.ZerodhaService, isProduction service.IsProduction, userSvc service.UserService) *ZerodhaController {
	return &ZerodhaController{zerodhaSvc: zerodhaSvc, isProduction: isProduction, userSvc: userSvc}
}

func (ctrl *ZerodhaController) RegisterRoutes(api huma.API) {
	authMw := middleware.HumaAuthMiddleware(api, bool(ctrl.isProduction))

	huma.Register(api, huma.Operation{
		OperationID: "zerodha-callback",
		Method:      http.MethodPost,
		Path:        "/api/zerodha/login",
		Summary:     "Handle Zerodha Callback",
		Description: "Exchanges the Zerodha request token for an access token and links it to the internal user.",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Zerodha"},
	}, ctrl.login)

	huma.Register(api, huma.Operation{
		OperationID: "get-zerodha-profile",
		Method:      http.MethodGet,
		Path:        "/api/zerodha/me",
		Summary:     "Get Zerodha profile for the authenticated user",
		Description: "Retrieves the authenticated user's details from the context and validates their Zerodha session/token status.",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Zerodha"},
	}, ctrl.auth)

	huma.Register(api, huma.Operation{
		OperationID: "zerodha-config",
		Method:      http.MethodPost,
		Path:        "/api/zerodha/config",
		Summary:     "Handle Zerodha Config",
		Description: "Stores the Zerodha config for the authenticated user.",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Zerodha"},
	}, ctrl.config)
}

func (ctrl *ZerodhaController) login(ctx context.Context, input *model.ZerodhaInput) (*model.ResponseWrapper, error) {
	token, err := ctrl.zerodhaSvc.GenerateAccessToken(input.Body.RequestToken, input.Body.UserId)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error Generating token")
	}
	cache.GoSet("zerodha_token_"+strconv.FormatInt(input.Body.UserId, 10), token, util.ZerodhaTokenExpiry())
	cache.KiteClientCache.Delete(strconv.FormatInt(input.Body.UserId, 10))
	return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Flow invocation success"}}, nil
}

func (ctrl *ZerodhaController) auth(ctx context.Context, input *struct{}) (*model.ResponseWrapper, error) {
	userDto, ok := ctx.Value("user").(model.UserDto)
	if !ok {
		return nil, huma.Error401Unauthorized("User context missing")
	}

	user, err := ctrl.userSvc.FindUser(ctx, 0, "", userDto.UserID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error getting user")
	}

	if user.ZerodhaConfig.ApiKey == "" || user.ZerodhaConfig.ApiSecret == "" {
		return nil, huma.Error404NotFound("E001")
	}

	var token string
	ok, err = database.RedisHelper.GetAsStruct("zerodha_token_"+strconv.FormatInt(user.UserID, 10), &token)
	if !ok || err != nil {
		return &model.ResponseWrapper{Body: model.Response{Success: false, Message: "Token expired", Data: user.ZerodhaConfig.ApiKey}}, nil
	}

	var kc *kiteconnect.Client
	val, ok := cache.KiteClientCache.Get(strconv.FormatInt(user.UserID, 10))
	if !ok {

		kc, err = ctrl.zerodhaSvc.InitiateKiteConnect(context.Background(), token, user.UserID)
		if err != nil {
			return &model.ResponseWrapper{Body: model.Response{Success: false, Message: "Token expired", Data: user.ZerodhaConfig.ApiKey}}, nil
		}
		cache.KiteClientCache.Set(strconv.FormatInt(user.UserID, 10), kc, util.ZerodhaTokenExpiry())
	} else {
		kc = val.(*kiteconnect.Client)
	}

	_, err = kc.GetUserProfile()
	if err != nil {
		return &model.ResponseWrapper{Body: model.Response{Success: false, Message: "Token expired", Data: user.ZerodhaConfig.ApiKey}}, nil
	}

	return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Token already exist", Data: user.UserID}}, nil
}

func (ctrl *ZerodhaController) config(ctx context.Context, input *struct{ Body model.ZerodhaConfig }) (*model.ResponseWrapper, error) {
	userDto, ok := ctx.Value("user").(model.UserDto)
	if !ok {
		return nil, huma.Error401Unauthorized("User context missing")
	}

	err := ctrl.userSvc.PatchUserData(ctx, userDto.UserID, model.User{
		ZerodhaConfig: input.Body,
	})

	if err != nil {
		return nil, huma.Error500InternalServerError("Error updating Zerodha configuration")
	}

	return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Zerodha configuration updated successfully", Data: userDto.UserID}}, nil
}
