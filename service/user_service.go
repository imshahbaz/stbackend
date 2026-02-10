package service

import (
	"context"
	"errors"
	"fmt"

	"backend/customerrors"
	"backend/model"
	"backend/repository"
	"backend/util"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type UserService interface {
	CreateUser(ctx context.Context, request model.UserDto) (*model.User, error)
	UpdateUserTheme(ctx context.Context, userId int64, theme model.UserTheme) (*model.User, error)
	UpdateUsername(ctx context.Context, userId int64, username string) (*model.User, error)
	GetNextSequence(ctx context.Context, sequenceName string) (int, error)
	FindUser(ctx context.Context, mobile int64, email string, userId int64) (*model.User, error)
	AddCredentials(ctx context.Context, userDto model.UserDto) (*model.User, error)
	PatchUserData(ctx context.Context, userId int64, user model.User) error
}

type UserServiceImpl struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) UserService {
	return &UserServiceImpl{repo: repo}
}

func (s *UserServiceImpl) CreateUser(ctx context.Context, request model.UserDto) (*model.User, error) {
	existing, err := s.FindUser(ctx, request.Mobile, request.Email, 0)

	if err != nil && !errors.Is(err, customerrors.ErrUserNotFound) {
		return nil, err
	}

	if existing != nil {
		return nil, customerrors.ErrUserAlreadyExists
	}

	password := request.Password
	if password == "" {
		password = util.GenerateRandomString(10)
	}

	user, err := request.ToEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to process user data: %w", err)
	}

	if user.Username == "" {
		user.Username = util.GenerateRandomString(10)
	}

	userId, err := s.GetNextSequence(ctx, "userid")
	if err != nil {
		return nil, fmt.Errorf("failed to generate user id: %w", err)
	}

	user.UserID = int64(userId)
	if err := s.repo.GenericRepo.Insert(ctx, *user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserServiceImpl) UpdateUserTheme(ctx context.Context, userId int64, theme model.UserTheme) (*model.User, error) {
	return s.repo.GenericRepo.UpdateFields(ctx, userId, bson.M{"theme": theme})
}

func (s *UserServiceImpl) UpdateUsername(ctx context.Context, userId int64, username string) (*model.User, error) {
	return s.repo.GenericRepo.UpdateFields(ctx, userId, bson.M{"username": username})
}

func (s *UserServiceImpl) GetNextSequence(ctx context.Context, sequenceName string) (int, error) {
	return s.repo.GetNextSequence(ctx, "userid")
}

func (s *UserServiceImpl) FindUser(ctx context.Context, mobile int64, email string, userId int64) (*model.User, error) {
	var orFilters []bson.M

	if userId <= 0 {
		if mobile > 0 {
			orFilters = append(orFilters, bson.M{"mobile": mobile})
		}
		if email != "" {
			orFilters = append(orFilters, bson.M{"email": email})
		}
	} else {
		orFilters = append(orFilters, bson.M{"_id": userId})
	}

	if len(orFilters) == 0 {
		return nil, customerrors.ErrUserNotFound
	}

	var user *model.User
	filter := bson.M{"$or": orFilters}

	user, err := s.repo.FindOneByFilter(ctx, filter)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, customerrors.ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func (s *UserServiceImpl) AddCredentials(ctx context.Context, userDto model.UserDto) (*model.User, error) {
	return s.repo.GenericRepo.UpdateFields(ctx, userDto.UserID, bson.M{
		"email":    userDto.Email,
		"password": userDto.Password,
	})
}

func (s *UserServiceImpl) PatchUserData(ctx context.Context, userId int64, user model.User) error {
	updateData := bson.M{}
	if user.Email != "" {
		updateData["email"] = user.Email
	}
	if user.Username != "" {
		updateData["username"] = user.Username
	}
	if user.Password != "" {
		updateData["password"] = user.Password
	}
	if user.Role != "" {
		updateData["role"] = user.Role
	}
	if user.Theme != "" {
		updateData["theme"] = user.Theme
	}
	if user.Mobile != 0 {
		updateData["mobile"] = user.Mobile
	}
	if user.Name != "" {
		updateData["name"] = user.Name
	}
	if user.Profile != "" {
		updateData["profile"] = user.Profile
	}
	if user.ZerodhaConfig.ApiKey != "" {
		updateData["zerodhaConfig"] = user.ZerodhaConfig
	}
	if user.MstockConfig.ApiKey != "" {
		updateData["mstockConfig"] = user.MstockConfig
	}

	_, err := s.repo.GenericRepo.UpdateFields(ctx, userId, updateData)
	return err
}
