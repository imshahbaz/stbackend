package customerrors

import "errors"

var (
	ErrUserAlreadyExists = errors.New("an account with this email already exists. Please log in")
	ErrUserNotFound      = errors.New("user not found")
)
