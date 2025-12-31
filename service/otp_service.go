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
	SendSignUpOtp(ctx context.Context, request model.SignupDto) error
	VerifyOtp(email, otp string) (bool, error)
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


func (s *OtpServiceImpl) SendSignUpOtp(ctx context.Context, request model.SignupDto) error {
	if _, found := localCache.OtpCache.Get(request.Email); found {
		return ErrDuplicateOtp
	}

	otp, err := util.GenerateOtp()
	if err != nil {
		return fmt.Errorf("failed to generate otp: %w", err)
	}

	emailRequest := s.buildSignupEmail(request.Email, otp)

	if err := s.emailService.SendEmail(ctx, emailRequest); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	localCache.OtpCache.Set(request.Email, otp, cache.DefaultExpiration)

	return nil
}

func (s *OtpServiceImpl) VerifyOtp(email, otp string) (bool, error) {
	cachedOtp, found := localCache.OtpCache.Get(email)
	if !found {
		return false, ErrInvalidOtp
	}

	if storedStr, ok := cachedOtp.(string); ok && storedStr == otp {
		localCache.OtpCache.Delete(email) // OTP is one-time use
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
