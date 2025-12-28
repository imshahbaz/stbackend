package model

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// --- ENUMS ---
// UserRole represents the account access level
// @Description ADMIN or USER access level
type UserRole string

// UserTheme represents the UI preference
// @Description LIGHT or DARK theme mode
type UserTheme string

const (
	RoleAdmin  UserRole  = "ADMIN"
	RoleUser   UserRole  = "USER"
	ThemeLight UserTheme = "LIGHT"
	ThemeDark  UserTheme = "DARK"
)

// --- MARGIN ---
// Margin represents the database entity for stock leverage
type Margin struct {
	Symbol string  `bson:"_id" json:"symbol"`
	Name   string  `bson:"name" json:"name"`
	Margin float32 `bson:"margin" json:"margin"`
}

// StockMarginDto combines stock price with margin requirements
type StockMarginDto struct {
	Name   string  `json:"name"`
	Symbol string  `json:"symbol"`
	Margin float32 `json:"margin"`
	Close  float32 `json:"close"`
}

// --- STRATEGY ---
// Strategy is the core scanner entity
type Strategy struct {
	Name       string `bson:"_id" json:"name"`
	ScanClause string `bson:"scanClause" json:"scanClause"`
	Active     bool   `bson:"active" json:"active"`
}

// StrategyDto is used for creating/updating strategies
type StrategyDto struct {
	Name       string `json:"name" validate:"required"`
	ScanClause string `json:"scanClause" validate:"required"`
	Active     bool   `json:"active"`
}

func (d *StrategyDto) ToEntity() Strategy {
	return Strategy{
		Name:       strings.ToUpper(d.Name),
		ScanClause: d.ScanClause,
		Active:     d.Active,
	}
}

// --- USER ---
// User is the main account entity
type User struct {
	Email    string    `bson:"_id" json:"email"` // Primary Key in MongoDB
	Username string    `bson:"username" json:"username"`
	Password string    `bson:"password" json:"password"`
	Role     UserRole  `bson:"role" json:"role"`
	Theme    UserTheme `bson:"theme" json:"theme"`
}

// ToDto maps the Entity to the API Response object
func (u *User) ToDto() UserDto {
	return UserDto{
		Email:    u.Email,
		Username: u.Username,
		Role:     u.Role,
		Theme:    u.Theme,
		// Password fields are left empty in the DTO
	}
}

// UserDto handles authentication requests
type UserDto struct {
	Email           string    `json:"email" validate:"required,email"`
	Username        string    `json:"username"`
	Password        string    `json:"password,omitempty"`
	ConfirmPassword string    `json:"confirmPassword,omitempty" validate:"required,eqfield=Password"`
	Role            UserRole  `json:"role"`
	Theme           UserTheme `json:"theme"`
}

func (d *UserDto) ToEntity() (*User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(d.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &User{
		Username: strings.ToLower(strings.Split(d.Email, "@")[0]),
		Email:    d.Email,
		Password: string(hashed),
		Role:     RoleUser,
		Theme:    ThemeDark,
	}, nil
}

// --- BREVO EMAIL ---
// BrevoEmailRequest is the payload for sending transactional emails
type Recipient struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type BrevoEmailRequest struct {
	Sender      Recipient   `json:"sender"`
	To          []Recipient `json:"to"`
	Subject     string      `json:"subject"`
	HTMLContent string      `json:"htmlContent"`
}

const SignupTemplate = `
<table style="max-width:400px;margin:auto;padding:20px;border:1px solid #ddd;border-radius:8px;">
  <tr><td style="font-family:Arial, sans-serif;">
    <p>Verification code: <h2 style="color:#1a73e8;">%s</h2></p>
    <p>Valid for %d minutes.</p>
  </td></tr>
</table>`

func (r *BrevoEmailRequest) Signup(otp string, validity int) {
	r.Subject = "Signup Verification Code"
	r.HTMLContent = fmt.Sprintf(SignupTemplate, otp, 5) // Use fmt.Sprintf if you want to inject values
}

// ChartInkResponseDto mimics the parent Java class
// ChartInkResponseDto maps the wrapper from ChartInk API
type ChartInkResponseDto struct {
	// json:"data" tells the parser to map the JSON key "data" to this field
	Data []StockData `json:"data"`
}

// StockData mimics the static inner class
// In Go, we usually keep them at the package level for better readability
// StockData represents a single row from a scan result
type StockData struct {
	NSECode string  `json:"nsecode"`
	Name    string  `json:"name"`
	Close   float32 `json:"close"`
}

// MessageResponse represents a standard JSON response for auth operations
// @Description Standard response containing status and a descriptive message
type MessageResponse struct {
	// OtpSent indicates if the OTP was successfully triggered
	OtpSent bool `json:"otpSent" example:"true"`

	// Message provides details about the operation result
	Message string `json:"message" example:"Otp sent successfully to user@example.com"`
}

type VerifyOtpRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
	Otp   string `json:"otp" binding:"required,len=6" example:"123456"`
}

// Common Response structure for all API calls
type Response struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"Update successful"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ObRequest struct {
	Symbol string  `json:"symbol"`
	Date   string  `json:"date"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
}

type ObResponse struct {
	StockMarginDto
	Date string `json:"date"`
}
