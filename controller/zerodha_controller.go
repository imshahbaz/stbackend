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
)

type ZerodhaController struct {
	zerodhaSvc   service.ZerodhaService
	isProduction bool
}

func NewZerodhaController(zerodhaSvc service.ZerodhaService, isProduction bool) *ZerodhaController {
	return &ZerodhaController{zerodhaSvc: zerodhaSvc, isProduction: isProduction}
}

func (ctrl *ZerodhaController) RegisterRoutes(api huma.API) {
	authMw := middleware.HumaAuthMiddleware(api, ctrl.isProduction)

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
}

func (ctrl *ZerodhaController) login(ctx context.Context, input *model.ZerodhaInput) (*model.ResponseWrapper, error) {
	token, err := ctrl.zerodhaSvc.GenerateAccessToken(input.Body.RequestToken, input.Body.UserId)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error Generating token")
	}
	cache.GoSet("zerodha_token_"+strconv.FormatInt(input.Body.UserId, 10), token, util.ZerodhaTokenExpiry())
	return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Flow invocation success"}}, nil
}

func (ctrl *ZerodhaController) auth(ctx context.Context, input *struct{}) (*model.ResponseWrapper, error) {
	user, ok := ctx.Value("user").(model.UserDto)
	if !ok {
		return nil, huma.Error401Unauthorized("User context missing")
	}

	ok = database.RedisHelper.Exists("zerodha_token_" + strconv.FormatInt(user.UserID, 10))
	if !ok {
		return nil, huma.Error404NotFound("Token expired")
	}

	return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Token already exist", Data: user.UserID}}, nil
}
