package controller

import (
	"backend/middleware"
	"backend/model"
	"backend/service"
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type MstockController struct {
	mstockSvc    service.MstockService
	isProduction bool
}

func NewMstockController(mstockSvc service.MstockService, isProduction bool) *MstockController {
	return &MstockController{mstockSvc: mstockSvc, isProduction: isProduction}
}

func (ctrl *MstockController) RegisterRoutes(api huma.API) {
	authMw := middleware.HumaAuthMiddleware(api, ctrl.isProduction)

	huma.Register(api, huma.Operation{
		OperationID: "mstock-login",
		Method:      http.MethodPost,
		Path:        "/api/mstock/login",
		Summary:     "Mstock Login",
		Description: "Initiates the login process for Mstock.",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Mstock"},
	}, ctrl.login)

	huma.Register(api, huma.Operation{
		OperationID: "mstock-verify-otp",
		Method:      http.MethodPost,
		Path:        "/api/mstock/verify",
		Summary:     "Mstock Verify OTP",
		Description: "Verifies the OTP for Mstock login.",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Mstock"},
	}, ctrl.verifyOtp)

	huma.Register(api, huma.Operation{
		OperationID: "mstock-place-order",
		Method:      http.MethodPost,
		Path:        "/api/mstock/order",
		Summary:     "Mstock Place Order",
		Description: "Places an order via Mstock.",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Mstock"},
	}, ctrl.placeOrder)

	huma.Register(api, huma.Operation{
		OperationID: "get-mstock-profile",
		Method:      http.MethodGet,
		Path:        "/api/mstock/me",
		Summary:     "Get Mstock profile for the authenticated user",
		Description: "Checks if the authenticated user has an active Mstock session.",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Mstock"},
	}, ctrl.auth)

	huma.Register(api, huma.Operation{
		OperationID: "mstock-refresh-token",
		Method:      http.MethodPost,
		Path:        "/api/mstock/refresh",
		Summary:     "Refresh Mstock Access Token",
		Description: "Uses stored user credentials to re-initiate m.Stock login.",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Mstock"},
	}, ctrl.refreshAccessToken)
}

func (ctrl *MstockController) login(ctx context.Context, input *struct{ Body model.MstockLoginInput }) (*model.ResponseWrapper, error) {
	user := ctx.Value("user").(model.UserDto)
	return ctrl.mstockSvc.Login(ctx, user.UserID, &input.Body)
}

func (ctrl *MstockController) verifyOtp(ctx context.Context, input *struct{ Body model.MstockVerifyOtpInput }) (*model.ResponseWrapper, error) {
	user := ctx.Value("user").(model.UserDto)
	return ctrl.mstockSvc.VerifyOtp(ctx, user.UserID, &input.Body)
}

func (ctrl *MstockController) placeOrder(ctx context.Context, input *struct{ Body model.MstockOrderRequest }) (*model.ResponseWrapper, error) {
	user := ctx.Value("user").(model.UserDto)
	return ctrl.mstockSvc.PlaceFnOrder(ctx, user.UserID, &input.Body)
}

func (ctrl *MstockController) auth(ctx context.Context, input *struct{}) (*model.ResponseWrapper, error) {
	userDto, ok := ctx.Value("user").(model.UserDto)
	if !ok {
		return nil, huma.Error401Unauthorized("User context missing")
	}

	return ctrl.mstockSvc.GetProfile(ctx, userDto.UserID)
}

func (ctrl *MstockController) refreshAccessToken(ctx context.Context, input *struct{}) (*model.ResponseWrapper, error) {
	userDto, ok := ctx.Value("user").(model.UserDto)
	if !ok {
		return nil, huma.Error401Unauthorized("User context missing")
	}

	return ctrl.mstockSvc.RefreshAccessToken(ctx, userDto.UserID)
}
