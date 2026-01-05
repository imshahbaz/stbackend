package service

import (
	"backend/cache"
	"backend/config"
	"backend/customerrors"
	"backend/model"
	"backend/util"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type GoogleAuthService interface {
	ValidateToken(ctx context.Context, input *model.AuthInput) (*model.GoogleAuthResponse, error)
}

type GoogleAuthServiceImpl struct {
	userSvc    UserService
	cfgManager *config.ConfigManager
}

func NewGoogleAuthService(userSvc UserService, cfgManager *config.ConfigManager) GoogleAuthService {
	return &GoogleAuthServiceImpl{userSvc: userSvc, cfgManager: cfgManager}
}

func (svc *GoogleAuthServiceImpl) ValidateToken(ctx context.Context, input *model.AuthInput) (*model.GoogleAuthResponse, error) {
	id := uuid.New().String()
	signUuid := util.SignState(id, svc.cfgManager.GetConfig().GoogleAuth.EncryptionKey)
	go func() {
		detachedCtx := context.Background()
		gUser, err := util.ValidateGoogleIDToken(detachedCtx, input.Code, svc.cfgManager.GetConfig().GoogleAuth.ClientID)
		if err != nil {
			log.Warn().Err(err).Msg("Invalid Google token")
			return
		}

		if gUser.Email == "" {
			return
		}

		user, err := svc.userSvc.FindUser(detachedCtx, 0, gUser.Email, 0)
		if err != nil && !errors.Is(err, customerrors.ErrUserNotFound) {
			log.Err(err).Msgf("Invalid Request %v", err)
			return
		}

		if user == nil {
			if err := svc.createUser(detachedCtx, *gUser, user); err != nil {
				return
			}
		}

		if gUser.Picture != "" && (user.Profile == "" || gUser.Picture != user.Profile) {
			if err := svc.userSvc.PatchUserData(detachedCtx, user.UserID, model.User{
				Profile: gUser.Picture,
			}); err != nil {
				log.Info().Msgf("Unable to update profile picture userId : %v", user.UserID)
			} else {
				user.Profile = gUser.Picture
			}
		}
		cache.GoSet("auth_"+id, user.ToDto(), 2*time.Minute)
	}()

	return &model.GoogleAuthResponse{
		Status: http.StatusOK,
		Body:   model.Response{Data: signUuid},
	}, nil
}

func (svc *GoogleAuthServiceImpl) createUser(ctx context.Context, gUser model.GoogleUser, user *model.User) error {
	dto := model.UserDto{
		Email:    gUser.Email,
		Username: gUser.GivenName + "_" + gUser.FamilyName,
		Role:     model.RoleUser,
		Theme:    model.ThemeDark,
		Name:     gUser.Name,
		Profile:  gUser.Picture,
	}

	newUser, err := svc.userSvc.CreateUser(ctx, dto)
	if err != nil {
		return fmt.Errorf("Invalid Request %v", err)
	}

	user = newUser
	return nil
}
