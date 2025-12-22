package model

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// --- ENUMS ---
type UserRole string
type UserTheme string

const (
	RoleAdmin  UserRole  = "ADMIN"
	RoleUser   UserRole  = "USER"
	ThemeLight UserTheme = "LIGHT"
	ThemeDark  UserTheme = "DARK"
)

// --- SYSTEM CONFIG ---
type EnvConfig struct {
	BrevoEmail    string `json:"brevoEmail"`
	BrevoApiKey   string `json:"brevoApiKey"`
	ApiKey        string `json:"apiKey"`
	MongoUser     string `json:"mongoUser"`
	MongoPassword string `json:"mongoPassword"`
}

// --- MARGIN ---
type Margin struct {
	Symbol string  `bson:"_id" json:"symbol"`
	Name   string  `bson:"name" json:"name"`
	Margin float32 `bson:"margin" json:"margin"`
}

type StockMarginDto struct {
	Name   string  `json:"name"`
	Symbol string  `json:"symbol"`
	Margin float32 `json:"margin"`
	Close  float32 `json:"close"`
}

// --- STRATEGY ---
type Strategy struct {
	Name       string `bson:"_id" json:"name"`
	ScanClause string `bson:"scanClause" json:"scanClause"`
	Active     bool   `bson:"active" json:"active"`
}

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
type User struct {
	Username string    `bson:"_id" json:"username"`
	Email    string    `bson:"email" json:"email"`
	Password string    `bson:"password" json:"-"`
	Role     UserRole  `bson:"role" json:"role"`
	Theme    UserTheme `bson:"theme" json:"theme"`
}

type UserDto struct {
	Email           string    `json:"email" validate:"required,email"`
	Password        string    `json:"password" validate:"required,min=8"`
	ConfirmPassword string    `json:"confirmPassword" validate:"required,eqfield=Password"`
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
		Role:     d.Role,
		Theme:    d.Theme,
	}, nil
}

// --- BREVO EMAIL ---
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
	r.HTMLContent = (SignupTemplate) // Use fmt.Sprintf if you want to inject values
}

// ChartInkResponseDto mimics the parent Java class
type ChartInkResponseDto struct {
	// json:"data" tells the parser to map the JSON key "data" to this field
	Data []StockData `json:"data"`
}

// StockData mimics the static inner class
// In Go, we usually keep them at the package level for better readability
type StockData struct {
	NSECode string  `json:"nsecode"`
	Name    string  `json:"name"`
	Close   float32 `json:"close"`
}
