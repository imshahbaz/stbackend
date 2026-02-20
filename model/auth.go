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

type SignupDto struct {
	LoginDto
	ConfirmPassword string `json:"confirmPassword" validate:"required,eqfield=Password" example:"secret"`
}

type TrueCallerStatusInput struct {
	RequestId string `path:"requestId"`
}

type AuthInput struct {
	Code  string `query:"code" doc:"The authorization code from Google"`
	State string `query:"state" doc:"Anti-forgery state token"`
}

type GoogleAuthResponse[T any] struct {
	Location  string `header:"Location"`
	SetCookie string `header:"Set-Cookie"`
	Status    int    `status:"302"`
	Body      Payload[T]
}

type ZerodhaLoginDto struct {
	RequestToken string `json:"request_token" doc:"The temporary request token returned by Zerodha after login" required:"true"`
	UserId       int64  `json:"user_id" doc:"The internal user identifier" required:"true"`
}
