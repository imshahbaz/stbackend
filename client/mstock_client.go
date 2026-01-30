package client

import (
	"backend/cache"
	"backend/model"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

type MstockClient struct {
	UserId         int64
	ApiKey         string
	AccessToken    string
	Client         *resty.Client
	MstockUserName string
}

func NewMstockClient(userId int64, apiKey string, mstockUserName string) *MstockClient {
	client := resty.New().
		SetBaseURL("https://api.mstock.trade/openapi/typea").
		SetTimeout(10 * time.Second)

	return &MstockClient{
		Client:         client,
		UserId:         userId,
		ApiKey:         apiKey,
		MstockUserName: mstockUserName,
	}
}

func (c *MstockClient) SetAccessToken(accessToken string) {
	c.AccessToken = accessToken
}

func (c *MstockClient) Login(input *model.MstockLoginInput) (*model.ResponseWrapper, error) {
	resp, err := c.Client.R().
		SetHeader("X-Mirae-Version", "1").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"username": input.Username,
			"password": input.Password,
		}).
		Post("/connect/login")

	if err != nil {
		log.Error().Err(err).Str("username", input.Username).Msg("MStock login connection failed")
		return nil, err
	}

	var result map[string]any
	sonic.Unmarshal(resp.Body(), &result)

	if status, ok := result["status"].(string); ok && status == "success" {
		cache.PendingLoginCache.Set(input.Username, input.APIKey, 5*time.Minute)
		log.Info().Str("username", input.Username).Msg("MStock login initiated (Step 1 success)")
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: true,
				Message: "OTP sent successfully",
				Data:    "S002",
			},
		}, nil
	}

	msg, _ := result["message"].(string)
	if msg == "" {
		msg = "m.Stock login failed without a specific reason."
	}
	log.Warn().Str("username", input.Username).Str("reason", msg).Msg("MStock login failed")

	return &model.ResponseWrapper{
		Body: model.Response{
			Success: false,
			Message: msg,
			Data:    "E002",
		},
	}, nil
}

func (c *MstockClient) Verify(input *model.MstockVerifyOtpInput) (*model.ResponseWrapper, error) {
	resp, err := c.Client.R().
		SetHeader("X-Mirae-Version", "1").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"api_key":       c.ApiKey,
			"request_token": input.Otp,
		}).
		Post("/session/token")

	if err != nil {
		log.Error().Err(err).Str("username", c.MstockUserName).Msg("MStock verification connection failed")
		return nil, err
	}

	var result map[string]any
	sonic.Unmarshal(resp.Body(), &result)

	if status, ok := result["status"].(string); ok && status == "success" {
		data, _ := result["data"].(map[string]any)
		log.Info().Str("username", c.MstockUserName).Msg("MStock session established successfully")
		c.SetAccessToken(data["access_token"].(string))
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: true,
				Message: "Session established successfully",
				Data:    "S002",
			},
		}, nil
	}

	msg, _ := result["message"].(string)
	if msg == "" {
		msg = "OTP verification failed."
	}
	log.Warn().Str("username", c.MstockUserName).Str("reason", msg).Msg("MStock verification failed")
	return &model.ResponseWrapper{
		Body: model.Response{
			Success: false,
			Message: msg,
			Data:    "E002",
		},
	}, nil
}

func (c *MstockClient) PlaceOrder(input *model.MstockOrderInput) (*model.ResponseWrapper, error) {
	variety := input.Variety
	if variety == "" {
		variety = "regular"
	}

	resp, err := c.Client.R().
		SetHeader("X-Mirae-Version", "1").
		SetHeader("Authorization", fmt.Sprintf("token %s:%s", c.ApiKey, c.AccessToken)).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"tradingsymbol":    input.Symbol,
			"exchange":         input.Exchange,
			"transaction_type": input.Side,
			"order_type":       input.Type,
			"quantity":         input.Qty,
			"product":          input.Product,
			"validity":         input.Validity,
			"price":            input.Price,
		}).
		Post("/orders/" + variety)

	if err != nil {
		log.Error().Err(err).Str("username", c.MstockUserName).Msg("MStock order placement connection failed")
		return nil, err
	}

	var finalResult map[string]any
	rawBody := resp.Body()

	var sliceResult []map[string]any
	if err := sonic.Unmarshal(rawBody, &sliceResult); err == nil && len(sliceResult) > 0 {
		finalResult = sliceResult[0]
	} else {
		if err := sonic.Unmarshal(rawBody, &finalResult); err != nil {
			log.Error().Err(err).Str("username", c.MstockUserName).Msg("Failed to unmarshal MStock response")
			return nil, err
		}
	}

	if status, ok := finalResult["status"].(string); ok && status == "success" {
		log.Info().Str("username", c.MstockUserName).Msg("MStock order placed successfully")
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: true,
				Message: "Order placed successfully",
				Data:    "S002",
			},
		}, nil
	}

	msg, _ := finalResult["message"].(string)
	if msg == "" {
		msg = "m.Stock order placement failed without a specific reason."
	}

	log.Warn().Str("username", c.MstockUserName).Str("reason", msg).Msg("MStock order placement failed")
	return &model.ResponseWrapper{
		Body: model.Response{
			Success: false,
			Message: msg,
			Data:    "E002",
		},
	}, nil
}
