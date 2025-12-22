package service

import (
	"context"
	"errors"
	"fmt"

	"backend/model"
	"backend/repository"
)

// Custom Errors to mimic Spring Exceptions
var (
	ErrUserAlreadyExists = errors.New("an account with this email already exists. Please log in")
	ErrUserNotFound      = errors.New("user not found")
)

// 1. Interface Definition
type UserService interface {
	CreateUser(ctx context.Context, request model.UserDto) (*model.User, error)
	UpdateUser(ctx context.Context, request model.UserDto) (*model.User, error)
	GetUser(ctx context.Context, email string) (*model.User, error)
	UpdateUserTheme(ctx context.Context, email string, theme model.UserTheme) (*model.User, error)
	UpdateUsername(ctx context.Context, email string, username string) (*model.User, error)
	DeleteUser(ctx context.Context, username string) error
}

// 2. Implementation Struct
type UserServiceImpl struct {
	repo *repository.UserRepository
}

// NewUserService replaces @RequiredArgsConstructor
func NewUserService(repo *repository.UserRepository) UserService {
	return &UserServiceImpl{repo: repo}
}

func (s *UserServiceImpl) CreateUser(ctx context.Context, request model.UserDto) (*model.User, error) {
	// 1. Check if user exists
	existing, err := s.repo.FindByEmail(ctx, request.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserAlreadyExists
	}

	// 2. Convert DTO to Entity (Hashing happens here)
	user, err := request.ToEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to process user data: %w", err)
	}

	// 3. Save to Repository
	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserServiceImpl) GetUser(ctx context.Context, email string) (*model.User, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserServiceImpl) UpdateUser(ctx context.Context, request model.UserDto) (*model.User, error) {
	// Re-uses GetUser logic to ensure existence
	user, err := s.GetUser(ctx, request.Email)
	if err != nil {
		return nil, err
	}

	// Update fields (excluding sensitive password unless provided)
	user.Role = request.Role
	user.Theme = request.Theme

	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserServiceImpl) UpdateUserTheme(ctx context.Context, email string, theme model.UserTheme) (*model.User, error) {
	user, err := s.GetUser(ctx, email)
	if err != nil {
		return nil, err
	}

	user.Theme = theme
	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserServiceImpl) UpdateUsername(ctx context.Context, email string, username string) (*model.User, error) {
	user, err := s.GetUser(ctx, email)
	if err != nil {
		return nil, err
	}

	user.Username = username
	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserServiceImpl) DeleteUser(ctx context.Context, username string) error {
	return s.repo.DeleteByUsername(ctx, username)
}
