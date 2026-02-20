package controller

import (
	"backend/cache"
	"backend/middleware"
	"backend/model"
	"backend/service"
	"backend/util"
	"context"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
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

func (ctrl *ZerodhaController) login(ctx context.Context, input *model.RequestBody[model.ZerodhaLoginDto]) (*model.TypedResponse[any], error) {
	token, err := ctrl.zerodhaSvc.GenerateAccessToken(input.Body.RequestToken, input.Body.UserId)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error Generating token: " + err.Error())
	}
	cache.GoSet("zerodha_token_"+strconv.FormatInt(input.Body.UserId, 10), token, util.ZerodhaTokenExpiry())
	cache.KiteClientCache.Delete(strconv.FormatInt(input.Body.UserId, 10))
	return NewTypedResponse[any](nil, "Flow invocation success"), nil
}

func (ctrl *ZerodhaController) auth(ctx context.Context, input *struct{}) (*model.TypedResponse[any], error) {
	userDto, ok := ctx.Value("user").(model.UserDto)
	if !ok {
		return nil, huma.Error401Unauthorized("User context missing")
	}

	user, err := ctrl.userSvc.FindUser(ctx, 0, "", userDto.UserID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error getting user: " + err.Error())
	}

	if user.ZerodhaConfig.ApiKey == "" || user.ZerodhaConfig.ApiSecret == "" {
		return nil, huma.Error404NotFound("E001")
	}

	kc, err := ctrl.zerodhaSvc.GetKiteClient(ctx, user.UserID)
	if err != nil {
		return &model.TypedResponse[any]{
			Body: model.Payload[any]{
				Success: false,
				Data:    user.ZerodhaConfig.ApiKey,
				Message: "Token expired",
			},
		}, nil
	}

	_, err = kc.GetUserProfile()
	if err != nil {
		return &model.TypedResponse[any]{
			Body: model.Payload[any]{
				Success: false,
				Data:    user.ZerodhaConfig.ApiKey,
				Message: "Token expired",
			},
		}, nil
	}

	return NewTypedResponse[any](user.UserID, "Token already exist"), nil
}

func (ctrl *ZerodhaController) config(ctx context.Context, input *model.RequestBody[model.ZerodhaConfig]) (*model.TypedResponse[int64], error) {
	userDto, ok := ctx.Value("user").(model.UserDto)
	if !ok {
		return nil, huma.Error401Unauthorized("User context missing")
	}

	err := ctrl.userSvc.PatchUserData(ctx, userDto.UserID, model.User{
		ZerodhaConfig: input.Body,
	})

	if err != nil {
		return nil, huma.Error500InternalServerError("Error updating Zerodha configuration: " + err.Error())
	}

	return NewTypedResponse(userDto.UserID, "Zerodha configuration updated successfully"), nil
}
