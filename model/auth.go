package model

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

// --- Huma Request/Response Structs ---

type LoginDto struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"secret"`
}

type LoginRequest struct {
	Body LoginDto
}

type LoginResponse struct {
	SetCookie string `header:"Set-Cookie"`
	Body      Response
}

type SignupDto struct {
	LoginDto
	ConfirmPassword string `json:"confirmPassword" validate:"required,eqfield=Password" example:"secret"`
}

type SignupRequest struct {
	Body SignupDto
}

// MessageResponseWrapper wraps a standard response for Huma
type MessageResponseWrapper struct {
	Body Response
}

type VerifyOtpInput struct {
	Body VerifyOtpRequest
}

type TrueCallerInput struct {
	Body TruecallerDto
}

type ResponseWrapper struct {
	Body Response
}

type TrueCallerStatusInput struct {
	RequestId string `path:"requestId"`
}

type DetailedResponseWrapper struct {
	SetCookie string `header:"Set-Cookie"`
	Body      Response
}

type LogoutResponse struct {
	SetCookie string `header:"Set-Cookie"`
	Body      Response
}
