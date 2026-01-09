package model

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/go-webauthn/webauthn/webauthn"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	UserID          int64                 `bson:"_id" json:"userId"`
	Email           string                `bson:"email" json:"email"`
	Username        string                `bson:"username" json:"username"`
	Password        string                `bson:"password" json:"password"`
	Role            UserRole              `bson:"role" json:"role"`
	Theme           UserTheme             `bson:"theme" json:"theme"`
	Mobile          int64                 `bson:"mobile" json:"mobile"`
	Name            string                `bson:"name" json:"name"`
	Profile         string                `bson:"profile" json:"profile"`
	Credentials     []webauthn.Credential `json:"credentials" bson:"credentials"`
	WebAuthnEnabled bool                  `json:"webAuthnEnabled" bson:"webAuthnEnabled"`
}

func (u *User) ToDto() UserDto {
	return UserDto{
		UserID:          u.UserID,
		Email:           u.Email,
		Username:        u.Username,
		Role:            u.Role,
		Theme:           u.Theme,
		Mobile:          u.Mobile,
		Name:            u.Name,
		Profile:         u.Profile,
		WebAuthnEnabled: u.WebAuthnEnabled,
		Credentials:     u.Credentials,
	}
}

type PasswordMatcher interface {
	GetPassword() string
	GetConfirm() string
}

type UserDto struct {
	UserID          int64                 `json:"userId"`
	Email           string                `json:"email" validate:"required,email"`
	Username        string                `json:"username"`
	Password        string                `json:"password,omitempty"`
	ConfirmPassword string                `json:"confirmPassword,omitempty" validate:"required,eqfield=Password"`
	Role            UserRole              `json:"role"`
	Theme           UserTheme             `json:"theme"`
	Mobile          int64                 `json:"mobile"`
	Name            string                `json:"name"`
	Profile         string                `json:"profile"`
	WebAuthnEnabled bool                  `json:"webAuthnEnabled"`
	Credentials     []webauthn.Credential `json:"credentials,omitempty"`
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
		UserID:          d.UserID,
		Username:        username,
		Email:           d.Email,
		Password:        string(hashed),
		Role:            RoleUser,
		Theme:           ThemeDark,
		Mobile:          d.Mobile,
		Name:            d.Name,
		Profile:         d.Profile,
		WebAuthnEnabled: d.WebAuthnEnabled,
		Credentials:     d.Credentials,
	}, nil
}

func (u *UserDto) GetPassword() string { return u.Password }
func (u *UserDto) GetConfirm() string  { return u.ConfirmPassword }

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

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
}

func (u *User) WebAuthnID() []byte { return []byte(strconv.FormatInt(u.UserID, 10)) }
func (u *User) WebAuthnName() string {
	if u.Email != "" {
		return u.Email
	}
	if u.Mobile != 0 {
		return strconv.FormatInt(u.Mobile, 10)
	}
	return fmt.Sprintf("User-%d", u.UserID)
}
func (u *User) WebAuthnDisplayName() string                { return u.Username }
func (u *User) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }

func (u *UserDto) WebAuthnID() []byte { return []byte(strconv.FormatInt(u.UserID, 10)) }
func (u *UserDto) WebAuthnName() string {
	if u.Email != "" {
		return u.Email
	}
	if u.Mobile != 0 {
		return strconv.FormatInt(u.Mobile, 10)
	}
	return fmt.Sprintf("User-%d", u.UserID)
}
func (u *UserDto) WebAuthnDisplayName() string                { return u.Username }
func (u *UserDto) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }

func (u *User) AddCredential(cred webauthn.Credential) {
	u.Credentials = append(u.Credentials, cred)
}

func (u *UserDto) AddCredential(cred webauthn.Credential) {
	u.Credentials = append(u.Credentials, cred)
}
