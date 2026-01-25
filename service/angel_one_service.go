package service

import (
	"backend/cache"
	"backend/database"
	"backend/model"
	"backend/util"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/pquerna/otp/totp"
	"github.com/rs/zerolog/log"
)

// import (
// 	"backend/cache"
// 	"backend/database"
// 	"backend/model"
// 	"backend/util"
// 	"time"

// 	"fmt"

// 	SmartApi "github.com/angel-one/smartapigo"
// 	"github.com/pquerna/otp/totp"
// 	"github.com/rs/zerolog/log"
// )

// var (
// 	angelOneClient *SmartApi.Client
// )

// type AngelOneService interface {
// 	RefreshBrokerSession() error
// 	GetLTP(tradingSymbol, symbolToken string) (float64, error)
// }

// type AngelOneServiceImpl struct {
// 	angelOneConfig *model.AngelOneConfig
// }

// func NewAngelOneService(angelOneConfig *model.AngelOneConfig) AngelOneService {
// 	return &AngelOneServiceImpl{angelOneConfig: angelOneConfig}
// }

// func GenerateTOTP(secret string) (string, error) {
// 	code, err := totp.GenerateCode(secret, time.Now())
// 	if err != nil {
// 		return "", err
// 	}
// 	return code, nil
// }

// func (s *AngelOneServiceImpl) RefreshBrokerSession() error {
// 	otp, _ := GenerateTOTP(s.angelOneConfig.Seed)

// 	angelOneClient = SmartApi.New(s.angelOneConfig.ClientID, s.angelOneConfig.Password, s.angelOneConfig.ApiKey)

// 	var accessToken string
// 	database.RedisHelper.GetAsStruct("broker_access_token", &accessToken)

// 	if accessToken != "" {
// 		angelOneClient.SetAccessToken(accessToken)
// 		return nil
// 	}

// 	session, err := angelOneClient.GenerateSession(otp)

// 	if err == nil {
// 		cache.GoSet("broker_access_token", session.AccessToken, util.ZerodhaTokenExpiry())
// 		log.Info().Msg("Broker session refreshed successfully")
// 		return nil
// 	} else {
// 		log.Error().Msg("Broker session refresh failed")
// 		return err
// 	}
// }

// func (s *AngelOneServiceImpl) GetLTP(tradingSymbol, symbolToken string) (float64, error) {
// 	if angelOneClient == nil {
// 		return 0, fmt.Errorf("angelOneClient not initialized, please refresh session first")
// 	}
// 	ltpResponse, err := angelOneClient.GetLTP(SmartApi.LTPParams{
// 		Exchange:      "NSE",
// 		TradingSymbol: tradingSymbol + "-EQ",
// 		SymbolToken:   symbolToken,
// 	})
// 	if err != nil {
// 		return 0, err
// 	}
// 	return ltpResponse.Ltp, nil
// }

type AngelOneService interface {
	RefreshBrokerSession() error
	GetLTP(tradingSymbol, symbolToken string) (float64, error)
}

type AngelOneServiceImpl struct {
	angelOneConfig *model.AngelOneConfig
	restyClient    *resty.Client
	token          string
}

func NewAngelOneService(angelOneConfig *model.AngelOneConfig) AngelOneService {
	client := resty.New().
		SetBaseURL("https://apiconnect.angelone.in").
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("X-UserType", "USER").
		SetHeader("X-SourceID", "WEB").
		SetHeader("X-PrivateKey", angelOneConfig.ApiKey).
		SetHeader("X-ClientLocalIP", "127.0.0.1").
		SetHeader("X-ClientPublicIP", "127.0.0.1").
		SetHeader("X-MACAddress", "00:00:00:00:00:00").
		SetTimeout(10 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(1 * time.Second)
	return &AngelOneServiceImpl{angelOneConfig: angelOneConfig, restyClient: client}
}

func generateTOTP(secret string) (string, error) {
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", err
	}
	return code, nil
}

func (s *AngelOneServiceImpl) RefreshBrokerSession() error {
	otp, err := generateTOTP(s.angelOneConfig.Seed)
	if err != nil {
		return fmt.Errorf("failed to generate TOTP: %w", err)
	}

	var accessToken string
	database.RedisHelper.GetAsStruct("broker_access_token", &accessToken)
	if accessToken != "" {
		s.token = accessToken
		return nil
	}

	var result model.AngelOneLoginResponse

	resp, err := s.restyClient.R().
		SetBody(map[string]string{
			"clientcode": s.angelOneConfig.ClientID,
			"password":   s.angelOneConfig.Password,
			"totp":       otp,
		}).
		SetResult(&result).
		Post("/rest/auth/angelbroking/user/v1/loginByPassword")

	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}

	if !resp.IsSuccess() || !result.Status {
		log.Error().Str("msg", result.Message).Msg("AngelOne login rejected")
		return fmt.Errorf("broker auth failed: %s", result.Message)
	}

	s.token = result.Data.JwtToken
	cache.GoSet("broker_access_token", result.Data.JwtToken, util.ZerodhaTokenExpiry())

	log.Info().Msg("Broker session established via Resty")
	return nil
}

func (s *AngelOneServiceImpl) GetLTP(tradingSymbol, symbolToken string) (float64, error) {
	var result model.QuoteResponse
	resp, err := s.restyClient.R().
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(map[string]any{
			"mode": "LTP",
			"exchangeTokens": map[string][]string{
				"NSE": {symbolToken},
			},
		}).
		SetResult(&result).
		Post("/rest/secure/angelbroking/market/v1/quote/")

	if err != nil {
		return 0, fmt.Errorf("resty request failed: %w", err)
	}

	if !resp.IsSuccess() || !result.Status {
		return 0, fmt.Errorf("api error: %s", result.Message)
	}

	if len(result.Data.Fetched) > 0 {
		return result.Data.Fetched[0].Ltp, nil
	}

	return 0, fmt.Errorf("no ltp data found for token %s", symbolToken)
}
