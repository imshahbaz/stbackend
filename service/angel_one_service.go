package service

import (
	"backend/cache"
	"backend/database"
	"backend/model"
	"backend/util"
	"time"

	"fmt"

	SmartApi "github.com/angel-one/smartapigo"
	"github.com/pquerna/otp/totp"
	"github.com/rs/zerolog/log"
)

var (
	angelOneClient *SmartApi.Client
)

type AngelOneService interface {
	RefreshBrokerSession() error
	GetLTP(tradingSymbol, symbolToken string) (float64, error)
}

type AngelOneServiceImpl struct {
	angelOneConfig *model.AngelOneConfig
}

func NewAngelOneService(angelOneConfig *model.AngelOneConfig) AngelOneService {
	return &AngelOneServiceImpl{angelOneConfig: angelOneConfig}
}

func GenerateTOTP(secret string) (string, error) {
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", err
	}
	return code, nil
}

func (s *AngelOneServiceImpl) RefreshBrokerSession() error {
	otp, _ := GenerateTOTP(s.angelOneConfig.Seed)

	angelOneClient = SmartApi.New(s.angelOneConfig.ClientID, s.angelOneConfig.Password, s.angelOneConfig.ApiKey)

	var accessToken string
	database.RedisHelper.GetAsStruct("broker_access_token", &accessToken)

	if accessToken != "" {
		angelOneClient.SetAccessToken(accessToken)
		return nil
	}

	session, err := angelOneClient.GenerateSession(otp)

	if err == nil {
		cache.GoSet("broker_access_token", session.AccessToken, util.ZerodhaTokenExpiry())
		log.Info().Msg("Broker session refreshed successfully")
		return nil
	} else {
		log.Error().Msg("Broker session refresh failed")
		return err
	}
}

func (s *AngelOneServiceImpl) GetLTP(tradingSymbol, symbolToken string) (float64, error) {
	if angelOneClient == nil {
		return 0, fmt.Errorf("angelOneClient not initialized, please refresh session first")
	}
	ltpResponse, err := angelOneClient.GetLTP(SmartApi.LTPParams{
		Exchange:      "NSE",
		TradingSymbol: tradingSymbol + "-EQ",
		SymbolToken:   symbolToken,
	})
	if err != nil {
		return 0, err
	}
	return ltpResponse.Ltp, nil
}
