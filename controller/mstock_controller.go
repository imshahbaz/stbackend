package controller

import (
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
	huma.Register(api, huma.Operation{
		OperationID: "mstock-login",
		Method:      http.MethodPost,
		Path:        "/api/mstock/login",
		Summary:     "Mstock Login",
		Description: "Initiates the login process for Mstock.",
		Tags:        []string{"Mstock"},
	}, ctrl.login)

	huma.Register(api, huma.Operation{
		OperationID: "mstock-verify-otp",
		Method:      http.MethodPost,
		Path:        "/api/mstock/verify",
		Summary:     "Mstock Verify OTP",
		Description: "Verifies the OTP for Mstock login.",
		Tags:        []string{"Mstock"},
	}, ctrl.verifyOtp)

	huma.Register(api, huma.Operation{
		OperationID: "mstock-place-order",
		Method:      http.MethodPost,
		Path:        "/api/mstock/order",
		Summary:     "Mstock Place Order",
		Description: "Places an order via Mstock.",
		Tags:        []string{"Mstock"},
	}, ctrl.placeOrder)
}

func (ctrl *MstockController) login(ctx context.Context, input *struct{ Body model.MstockLoginInput }) (*model.ResponseWrapper, error) {
	return ctrl.mstockSvc.Login(ctx, &input.Body)
}

func (ctrl *MstockController) verifyOtp(ctx context.Context, input *struct{ Body model.MstockVerifyOtpInput }) (*model.ResponseWrapper, error) {
	return ctrl.mstockSvc.VerifyOtp(ctx, &input.Body)
}

func (ctrl *MstockController) placeOrder(ctx context.Context, input *struct{ Body model.MstockOrderInput }) (*model.ResponseWrapper, error) {
	return ctrl.mstockSvc.PlaceFnOrder(ctx, &input.Body)
}
