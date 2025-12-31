package model

import (
	"math/rand/v2"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	UserID   int64     `bson:"_id" json:"userId"`
	Email    string    `bson:"email" json:"email"`
	Username string    `bson:"username" json:"username"`
	Password string    `bson:"password" json:"password"`
	Role     UserRole  `bson:"role" json:"role"`
	Theme    UserTheme `bson:"theme" json:"theme"`
	Mobile   int64     `bson:"mobile" json:"mobile"`
	Name     string    `bson:"name" json:"name"`
}

func (u *User) ToDto() UserDto {
	return UserDto{
		UserID:   u.UserID,
		Email:    u.Email,
		Username: u.Username,
		Role:     u.Role,
		Theme:    u.Theme,
		Mobile:   u.Mobile,
		Name:     u.Name,
	}
}

type UserDto struct {
	UserID          int64     `json:"userId"`
	Email           string    `json:"email" validate:"required,email"`
	Username        string    `json:"username"`
	Password        string    `json:"password,omitempty"`
	ConfirmPassword string    `json:"confirmPassword,omitempty" validate:"required,eqfield=Password"`
	Role            UserRole  `json:"role"`
	Theme           UserTheme `json:"theme"`
	Mobile          int64     `json:"mobile"`
	Name            string    `json:"name"`
}

func (d *UserDto) ToEntity() (*User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(d.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var username string
	if d.Email != "" {
		username = strings.ToLower(strings.Split(d.Email, "@")[0])
	} else if d.Name != "" {
		username = strings.ToLower(strings.Split(d.Name, " ")[0] + strconv.Itoa(rand.IntN(10)+1))
	}

	return &User{
		UserID:   d.UserID,
		Username: username,
		Email:    d.Email,
		Password: string(hashed),
		Role:     RoleUser,
		Theme:    ThemeDark,
		Mobile:   d.Mobile,
		Name:     d.Name,
	}, nil
}

type UpdateThemeRequest struct {
	Theme UserTheme `json:"theme"`
}

type UpdateUsernameInput struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
}

type UpdateUsernameRequest struct {
	Body UpdateUsernameInput
}

type UpdateThemeInput struct {
	Body UpdateThemeRequest
}
