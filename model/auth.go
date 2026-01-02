package model

type UserRole string

type UserTheme string

const (
	RoleAdmin  UserRole  = "ADMIN"
	RoleUser   UserRole  = "USER"
	ThemeLight UserTheme = "LIGHT"
	ThemeDark  UserTheme = "DARK"
)

type MessageResponse struct {
	OtpSent bool `json:"otpSent" example:"true"`

	Message string `json:"message" example:"Otp sent successfully to user@example.com"`
}

type VerifyOtpRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
	Otp   string `json:"otp" binding:"required,len=6" example:"123456"`
}

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

type AuthInput struct {
	Code  string `query:"code" doc:"The authorization code from Google"`
	State string `query:"state" doc:"Anti-forgery state token"`
}

type GoogleAuthResponse struct {
	Location  string `header:"Location"`
	SetCookie string `header:"Set-Cookie"`
	Status    int    `status:"302"`
	Body      Response
}
