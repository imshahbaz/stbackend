package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	localCache "backend/cache"
	"backend/config"
	"backend/model"
	"backend/util"

	"github.com/patrickmn/go-cache"
)

// --- 1. Custom Errors ---
var (
	ErrDuplicateOtp = errors.New("OTP already sent. Please wait until it expires (5 minutes)")
	ErrInvalidOtp   = errors.New("invalid OTP. Please try again")
)

// --- 2. Interface Definition ---
type OtpService interface {
	SendSignUpOtp(ctx context.Context, request model.UserDto) error
	VerifyOtp(email, otp string) (bool, error)
}

// --- 3. Implementation Struct ---
type OtpServiceImpl struct {
	emailService EmailService
	cfg          *config.ConfigManager
}

// NewOtpService replaces @RequiredArgsConstructor
func NewOtpService(emailService EmailService, cfg *config.ConfigManager) OtpService {
	return &OtpServiceImpl{
		emailService: emailService,
		cfg:          cfg,
	}
}

// --- 4. Service Methods ---

// SendSignUpOtp handles generating, sending, and caching the registration OTP.
func (s *OtpServiceImpl) SendSignUpOtp(ctx context.Context, request model.UserDto) error {
	// 1. Rate Limit: Check if an OTP is already active in cache
	if _, found := localCache.OtpCache.Get(request.Email); found {
		return ErrDuplicateOtp
	}

	// 2. Generate secure OTP
	otp, err := util.GenerateOtp()
	if err != nil {
		return fmt.Errorf("failed to generate otp: %w", err)
	}

	// 3. Construct Brevo Email Request
	emailRequest := s.buildSignupEmail(request.Email, otp)

	// 4. Send via Email Service
	if err := s.emailService.SendEmail(ctx, emailRequest); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	// 5. Commit to Cache with 5-minute expiration
	localCache.OtpCache.Set(request.Email, otp, cache.DefaultExpiration)

	return nil
}

// VerifyOtp checks if the provided OTP matches the cached value.
func (s *OtpServiceImpl) VerifyOtp(email, otp string) (bool, error) {
	cachedOtp, found := localCache.OtpCache.Get(email)
	if !found {
		return false, ErrInvalidOtp
	}

	// Type assertion and comparison
	if storedStr, ok := cachedOtp.(string); ok && storedStr == otp {
		localCache.OtpCache.Delete(email) // OTP is one-time use
		return true, nil
	}

	return false, ErrInvalidOtp
}

// --- 5. Internal Helpers ---

// buildSignupEmail encapsulates the mapping logic for the email model.
func (s *OtpServiceImpl) buildSignupEmail(email, otp string) model.BrevoEmailRequest {
	userName := strings.Split(email, "@")[0]
	conf := s.cfg.GetConfig()

	req := model.BrevoEmailRequest{
		Sender: model.Recipient{
			Email: conf.BrevoEmail,
			Name:  "Shahbaz Trades",
		},
		To: []model.Recipient{
			{
				Email: email,
				Name:  userName,
			},
		},
	}

	// Apply signup template logic (setting subject and html content)
	req.Signup(otp, 5)
	return req
}
