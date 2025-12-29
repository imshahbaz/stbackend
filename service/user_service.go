package service

import (
	"context"
	"errors"
	"fmt"

	"backend/model"
	"backend/repository"
)

// --- 1. Custom Errors ---
// These mimic Spring-style Global Exception handling for specific business logic.
var (
	ErrUserAlreadyExists = errors.New("an account with this email already exists. Please log in")
	ErrUserNotFound      = errors.New("user not found")
)

// --- 2. Interface Definition ---
type UserService interface {
	CreateUser(ctx context.Context, request model.UserDto) (*model.User, error)
	UpdateUser(ctx context.Context, request model.UserDto) (*model.User, error)
	GetUser(ctx context.Context, email string) (*model.User, error)
	UpdateUserTheme(ctx context.Context, email string, theme model.UserTheme) (*model.User, error)
	UpdateUsername(ctx context.Context, email string, username string) (*model.User, error)
	DeleteUser(ctx context.Context, username string) error
}

// --- 3. Implementation Struct ---
type UserServiceImpl struct {
	repo *repository.UserRepository
}

// NewUserService initializes the implementation (Constructor Injection)
func NewUserService(repo *repository.UserRepository) UserService {
	return &UserServiceImpl{repo: repo}
}

// --- 4. Core Service Methods ---

// CreateUser handles registration logic: Check Existence -> Map to Entity -> Save
func (s *UserServiceImpl) CreateUser(ctx context.Context, request model.UserDto) (*model.User, error) {
	// Directly check repo to prevent the helper from returning ErrUserNotFound incorrectly
	existing, err := s.repo.FindByEmail(ctx, request.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserAlreadyExists
	}

	// Conversion (Hashing logic should be inside ToEntity as per your current setup)
	user, err := request.ToEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to process user data: %w", err)
	}

	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUser retrieves a user or returns a clean domain error
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

// UpdateUser updates general profile fields
func (s *UserServiceImpl) UpdateUser(ctx context.Context, request model.UserDto) (*model.User, error) {
	return s.applyUpdate(ctx, request.Email, func(user *model.User) {
		user.Role = request.Role
		user.Theme = request.Theme
		if request.Username != "" {
			user.Username = request.Username
		}
	})
}

// UpdateUserTheme updates only the UI theme preference
func (s *UserServiceImpl) UpdateUserTheme(ctx context.Context, email string, theme model.UserTheme) (*model.User, error) {
	return s.applyUpdate(ctx, email, func(user *model.User) {
		user.Theme = theme
	})
}

// UpdateUsername updates the user's display name
func (s *UserServiceImpl) UpdateUsername(ctx context.Context, email string, username string) (*model.User, error) {
	return s.applyUpdate(ctx, email, func(user *model.User) {
		user.Username = username
	})
}

// DeleteUser removes the user from the repository
func (s *UserServiceImpl) DeleteUser(ctx context.Context, username string) error {
	return s.repo.DeleteByUsername(ctx, username)
}

// --- 5. Internal Persistence Helper ---

// applyUpdate consolidates the Fetch -> Modify -> Save lifecycle.
// It ensures that we don't repeat error checking and repository calls in every update method.
func (s *UserServiceImpl) applyUpdate(ctx context.Context, email string, updateFn func(*model.User)) (*model.User, error) {
	// 1. Fetch
	user, err := s.GetUser(ctx, email)
	if err != nil {
		return nil, err
	}

	// 2. Modify via closure
	updateFn(user)

	// 3. Persist
	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
