package controller

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"backend/cache"
	"backend/customerrors"
	"backend/database"
	"backend/middleware"
	"backend/model"
	"backend/service"
	"backend/validator"

	"github.com/Oudwins/zog"
	"github.com/danielgtaylor/huma/v2"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/bcrypt"
)

type UserController struct {
	userSvc      service.UserService
	isProduction bool
	otpSvc       service.OtpService
}

func NewUserController(s service.UserService, isProduction bool, otpSvc service.OtpService) *UserController {
	return &UserController{userSvc: s, isProduction: isProduction, otpSvc: otpSvc}
}

func (ctrl *UserController) RegisterRoutes(api huma.API) {
	authMw := middleware.HumaAuthMiddleware(api, ctrl.isProduction)

	huma.Register(api, huma.Operation{
		OperationID: "update-username",
		Method:      http.MethodPatch,
		Path:        "/api/user/username",
		Summary:     "Update Username",
		Description: "Updates the username and invalidates the auth cache",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"User"},
	}, ctrl.UpdateUsername)

	huma.Register(api, huma.Operation{
		OperationID: "update-theme",
		Method:      http.MethodPatch,
		Path:        "/api/user/theme",
		Summary:     "Update User Theme",
		Description: "Updates preference (LIGHT/DARK) for the authenticated user",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"User"},
	}, ctrl.UpdateTheme)

	huma.Register(api, huma.Operation{
		OperationID: "send-update-otp",
		Method:      http.MethodPost,
		Path:        "/api/user/send-update-otp",
		Summary:     "Send OTP for Updating Credentials",
		Description: "Sends an OTP to the user's email for verifying credential updates",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"User"},
	}, ctrl.sendUpdateOtp)

	huma.Register(api, huma.Operation{
		OperationID: "verify-update-otp",
		Method:      http.MethodPost,
		Path:        "/api/user/verify-update-otp",
		Summary:     "Verify OTP for Updating Credentials",
		Description: "Verifies the OTP and updates the user's credentials upon successful verification",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"User"},
	}, ctrl.verifyUpdateOtp)
}

func (ctrl *UserController) UpdateUsername(ctx context.Context, input *model.UpdateUsernameRequest) (*model.DefaultResponse, error) {
	req := input.Body

	ctxt, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := ctrl.userSvc.UpdateUsername(ctxt, req.UserID, req.Username)
	if err != nil {
		return NewErrorResponse("Failed to update username"), nil
	}

	database.RedisHelper.Delete("auth_" + strconv.FormatInt(req.UserID, 10))

	return NewResponse(nil, "Username updated successfully"), nil
}

func (ctrl *UserController) UpdateTheme(ctx context.Context, input *model.UpdateThemeInput) (*model.DefaultResponse, error) {
	req := input.Body

	if req.Theme != model.ThemeLight && req.Theme != model.ThemeDark {
		return nil, huma.Error400BadRequest("Invalid theme: must be LIGHT or DARK")
	}

	val := ctx.Value("user")
	if val == nil {
		return nil, huma.Error401Unauthorized("User session not found")
	}

	userDto := val.(model.UserDto)

	ctxt, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := ctrl.userSvc.UpdateUserTheme(ctxt, userDto.UserID, req.Theme); err != nil {
		return NewErrorResponse("Internal server error"), nil
	}

	database.RedisHelper.Delete("auth_" + strconv.FormatInt(userDto.UserID, 10))

	return NewResponse(req.Theme, "Theme synchronized"), nil
}

func (ctrl *UserController) sendUpdateOtp(ctx context.Context, input *model.Request) (*model.MessageResponseWrapper, error) {
	var req model.UserDto
	if err := mapstructure.Decode(input.Body, &req); err != nil {
		return nil, huma.Error400BadRequest("Invalid Request")
	}

	authUser := ctx.Value("user").(model.UserDto)
	if authUser.UserID != req.UserID {
		return nil, huma.Error403Forbidden("Unauthorized to update credentials for this user")
	}

	bodyValidation := zog.Struct(validator.UserIdShape).
		Extend(validator.BaseShape).
		Extend(validator.PasswordShape).
		Extend(validator.ConfirmShape).
		TestFunc(validator.PasswordMatchTest)

	if err := bodyValidation.Validate(&req); err != nil {
		log.Printf("Validation error %v", err)
		return nil, huma.Error400BadRequest("Invalid Request")
	}

	existingUser, err := ctrl.userSvc.FindUser(ctx, 0, req.Email, 0)
	if err != nil && !errors.Is(err, customerrors.ErrUserNotFound) {
		return nil, huma.Error500InternalServerError("Unable to process request at this time")
	}

	if existingUser != nil {
		return nil, huma.Error400BadRequest("Email already in use")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, huma.Error500InternalServerError("Something went wrong")
	}
	req.Password = string(hashed)
	cache.SetUserCache(strconv.FormatInt(req.UserID, 10), req, model.CredUpdate)
	ctxt, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := ctrl.otpSvc.SendOtp(ctxt, req.Email, model.OTPUpdate); err != nil {
		if errors.Is(err, service.ErrDuplicateOtp) {
			return nil, huma.Error409Conflict(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &model.MessageResponseWrapper{
		Body: model.Response{
			Success: true,
			Message: "OTP sent to " + req.Email,
			Data:    model.MessageResponse{OtpSent: true, Message: "OTP sent to " + req.Email},
		},
	}, nil
}

func (ctrl *UserController) verifyUpdateOtp(ctx context.Context, input *model.VerifyOtpInput) (*model.MessageResponseWrapper, error) {
	req := input.Body
	authUser := ctx.Value("user").(model.UserDto)

	match, err := ctrl.otpSvc.VerifyOtp(req.Email, req.Otp, model.OTPUpdate)
	if err != nil || !match {
		return nil, huma.Error400BadRequest("Invalid OTP")
	}

	var cacheUser model.UserDto
	ok, err := cache.GetUserCache(strconv.FormatInt(authUser.UserID, 10), &cacheUser, model.CredUpdate)
	if err != nil || !ok {
		return nil, huma.Error400BadRequest("Invalid or expired request")
	}

	_, err = ctrl.userSvc.AddCredentials(ctx, cacheUser)
	if err != nil {
		log.Printf("Error adding credentials: %v", err)
		return nil, huma.Error500InternalServerError("Something went wrong")
	}

	cache.DeleteUserCache(req.Email, model.CredUpdate)
	database.RedisHelper.Delete("auth_" + strconv.FormatInt(authUser.UserID, 10))
	return &model.MessageResponseWrapper{Body: model.Response{Success: true, Message: "Credential added successfully"}}, nil
}
