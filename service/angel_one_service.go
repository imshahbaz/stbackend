package service

import (
	"backend/cache"
	"backend/database"
	"backend/model"
	"backend/util"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/go-resty/resty/v2"
	"github.com/pquerna/otp/totp"
	"github.com/rs/zerolog/log"
)

const (
	IntervalOneMin     = "ONE_MINUTE"
	IntervalThreeMin   = "THREE_MINUTE"
	IntervalFiveMin    = "FIVE_MINUTE"
	IntervalTenMin     = "TEN_MINUTE"
	IntervalFifteenMin = "FIFTEEN_MINUTE"
	IntervalThirtyMin  = "THIRTY_MINUTE"
	IntervalOneHour    = "ONE_HOUR"
	IntervalOneDay     = "ONE_DAY"
)

type AngelOneService interface {
	RefreshBrokerSession() error
	GetLTP(tradingSymbol, symbolToken string) (float64, error)
	GetMultipleLTP(tokens []string) (map[string]float64, error)
	GetHistoricalData(symbolToken string, interval string, fromDate, toDate string) ([]model.AngelOneCandle, error)
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
		SetRetryWaitTime(1 * time.Second).
		SetJSONMarshaler(sonic.ConfigDefault.Marshal).
		SetJSONUnmarshaler(sonic.ConfigDefault.Unmarshal)

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
	var accessToken string
	database.RedisHelper.GetAsStruct("broker_access_token", &accessToken)
	if accessToken != "" {
		s.token = accessToken
		return nil
	}

	otp, err := generateTOTP(s.angelOneConfig.Seed)
	if err != nil {
		return fmt.Errorf("failed to generate TOTP: %w", err)
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

func (s *AngelOneServiceImpl) GetMultipleLTP(tokens []string) (map[string]float64, error) {
	var result model.QuoteResponse

	requestBody := map[string]any{
		"mode": "LTP",
		"exchangeTokens": map[string][]string{
			"NSE": tokens,
		},
	}

	resp, err := s.restyClient.R().
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(requestBody).
		SetResult(&result).
		Post("/rest/secure/angelbroking/market/v1/quote/")

	if err != nil {
		return nil, fmt.Errorf("bulk ltp request failed: %w", err)
	}

	if !resp.IsSuccess() || !result.Status {
		return nil, fmt.Errorf("api error: %s (code: %s)", result.Message, result.Errorcode)
	}

	ltpMap := make(map[string]float64, len(result.Data.Fetched))
	for _, item := range result.Data.Fetched {
		ltpMap[item.SymbolToken] = item.Ltp
	}

	return ltpMap, nil
}

func (s *AngelOneServiceImpl) GetHistoricalData(symbolToken string, interval string, fromDate, toDate string) ([]model.AngelOneCandle, error) {
	var result model.CandleDataResponse

	resp, err := s.restyClient.R().
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(map[string]string{
			"exchange":    "NSE",
			"symboltoken": symbolToken,
			"interval":    interval,
			"fromdate":    fromDate,
			"todate":      toDate,
		}).
		SetResult(&result).
		Post("/rest/secure/angelbroking/historical/v1/getCandleData")

	if err != nil {
		return nil, fmt.Errorf("historical data request failed: %w", err)
	}

	if !resp.IsSuccess() || !result.Status {
		return nil, fmt.Errorf("api error: %s", result.Message)
	}

	var candles []model.AngelOneCandle
	for _, row := range result.Data {
		if len(row) < 6 {
			continue
		}

		candle := model.AngelOneCandle{
			Time:   fmt.Sprint(row[0]),
			Open:   row[1].(float64),
			High:   row[2].(float64),
			Low:    row[3].(float64),
			Close:  row[4].(float64),
			Volume: int64(row[5].(float64)),
		}
		candles = append(candles, candle)
	}

	return candles, nil
}
