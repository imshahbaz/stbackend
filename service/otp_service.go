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

var (
	ErrDuplicateOtp = errors.New("OTP already sent. Please wait until it expires (5 minutes)")
	ErrInvalidOtp   = errors.New("invalid OTP. Please try again")
)

type OtpService interface {
	SendOtp(ctx context.Context, email string, otpType model.OTPType) error
	VerifyOtp(email, otp string, otpType model.OTPType) (bool, error)
}

type OtpServiceImpl struct {
	emailService EmailService
	cfg          *config.ConfigManager
}

func NewOtpService(emailService EmailService, cfg *config.ConfigManager) OtpService {
	return &OtpServiceImpl{
		emailService: emailService,
		cfg:          cfg,
	}
}

func (s *OtpServiceImpl) SendOtp(ctx context.Context, email string, otpType model.OTPType) error {
	cacheKey, err := s.otpCacheKey(email, otpType)
	if err != nil {
		return fmt.Errorf("failed to generate otp: %w", err)
	}

	if _, found := localCache.OtpCache.Get(cacheKey); found {
		return ErrDuplicateOtp
	}

	otp, err := util.GenerateOtp()
	if err != nil {
		return fmt.Errorf("failed to generate otp: %w", err)
	}

	emailRequest, err := s.getEmailRequest(email, otp, otpType)
	if err != nil {
		return fmt.Errorf("failed to build email request: %w", err)
	}

	if err := s.emailService.SendEmail(ctx, *emailRequest); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	localCache.OtpCache.Set(cacheKey, otp, cache.DefaultExpiration)

	return nil
}

func (s *OtpServiceImpl) VerifyOtp(email, otp string, otpType model.OTPType) (bool, error) {
	cacheKey, err := s.otpCacheKey(email, otpType)
	if err != nil {
		return false, ErrInvalidOtp
	}
	cachedOtp, found := localCache.OtpCache.Get(cacheKey)
	if !found {
		return false, ErrInvalidOtp
	}

	if storedStr, ok := cachedOtp.(string); ok && storedStr == otp {
		localCache.OtpCache.Delete(cacheKey)
		return true, nil
	}

	return false, ErrInvalidOtp
}

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

	req.Signup(otp, 5)
	return req
}

func (s *OtpServiceImpl) otpCacheKey(email string, otpType model.OTPType) (string, error) {
	var cacheKey string
	switch otpType {
	case model.OTPRegister:
		cacheKey = email
	case model.OTPUpdate:
		cacheKey = email + "_update"
	default:
		return "", fmt.Errorf("invalid OTP type: %s", otpType)
	}

	return cacheKey, nil
}

func (s *OtpServiceImpl) buildUpdateEmail(email, otp string) model.BrevoEmailRequest {
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

	req.EmailVerification(otp, 5)
	return req
}

func (s *OtpServiceImpl) getEmailRequest(email, otp string, otpType model.OTPType) (*model.BrevoEmailRequest, error) {
	var emailRequest *model.BrevoEmailRequest

	switch otpType {
	case model.OTPRegister:
		req := s.buildSignupEmail(email, otp)
		emailRequest = &req
	case model.OTPUpdate:
		req := s.buildUpdateEmail(email, otp)
		emailRequest = &req
	default:
		return nil, fmt.Errorf("unsupported OTP type for email: %s", otpType)
	}

	return emailRequest, nil
}
