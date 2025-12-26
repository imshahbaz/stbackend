package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/model"
	"backend/util"

	"github.com/patrickmn/go-cache"
)

// Define custom errors to mimic your Java Exceptions
var (
	ErrDuplicateOtp = errors.New("OTP already sent. Please wait until it expires (5 minutes)")
	ErrInvalidOtp   = errors.New("invalid OTP. Please try again")
)

// 1. Interface Definition
type OtpService interface {
	SendSignUpOtp(ctx context.Context, request model.UserDto) error
	VerifyOtp(email, otp string) (bool, error)
}

// 2. Implementation Struct
type OtpServiceImpl struct {
	emailService EmailService
	otpCache     *cache.Cache
	brevoEmail   string
}

// NewOtpService replaces @RequiredArgsConstructor
func NewOtpService(emailService EmailService, brevoEmail string) OtpService {
	return &OtpServiceImpl{
		emailService: emailService,
		otpCache:     cache.New(5*time.Minute, 10*time.Minute),
		brevoEmail:   brevoEmail,
	}
}

func (s *OtpServiceImpl) SendSignUpOtp(ctx context.Context, request model.UserDto) error {
	// 1. Check if OTP exists in cache
	if _, found := s.otpCache.Get(request.Email); found {
		return ErrDuplicateOtp
	}

	// 2. Generate OTP using our secure utility
	otp, err := util.GenerateOtp()
	if err != nil {
		return fmt.Errorf("failed to generate otp: %w", err)
	}

	// 3. Construct Brevo Request
	userName := strings.Split(request.Email, "@")[0]
	emailRequest := model.BrevoEmailRequest{
		Sender: model.Recipient{
			Email: s.brevoEmail,
			Name:  "Shahbaz Trades",
		},
		To: []model.Recipient{
			{
				Email: request.Email,
				Name:  userName,
			},
		},
	}
	// Use the method we defined in model/user.go
	emailRequest.Signup(otp, 5)

	// 4. Send Email
	if err := s.emailService.SendEmail(ctx, emailRequest); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	// 5. Store in cache with 5-minute expiration
	s.otpCache.Set(request.Email, otp, cache.DefaultExpiration)

	return nil
}

func (s *OtpServiceImpl) VerifyOtp(email, otp string) (bool, error) {
	cachedOtp, found := s.otpCache.Get(email)

	if found && cachedOtp.(string) == otp {
		// Equivalent to invalidate
		s.otpCache.Delete(email)
		return true, nil
	}

	return false, ErrInvalidOtp
}
