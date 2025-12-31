package controller

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"backend/cache"
	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type UserController struct {
	userSvc      service.UserService
	isProduction bool
}

func NewUserController(s service.UserService, isProduction bool) *UserController {
	return &UserController{userSvc: s, isProduction: isProduction}
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
}

func (ctrl *UserController) UpdateUsername(ctx context.Context, input *model.UpdateUsernameRequest) (*model.DefaultResponse, error) {
	req := input.Body

	ctxt, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := ctrl.userSvc.UpdateUsername(ctxt, req.UserID, req.Username)
	if err != nil {
		return NewErrorResponse("Failed to update username"), nil
	}

	cache.UserAuthCache.Delete(strconv.FormatInt(req.UserID, 10))

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

	cache.UserAuthCache.Delete(strconv.FormatInt(userDto.UserID, 10))

	return NewResponse(req.Theme, "Theme synchronized"), nil
}
