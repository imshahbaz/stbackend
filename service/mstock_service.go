package service

import (
	"backend/cache"
	"backend/client"
	"backend/database"
	"backend/model"
	"backend/util"
	"context"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"
)

type MstockService interface {
	Login(ctx context.Context, userId int64, input *model.MstockLoginInput) (*model.ResponseWrapper, error)
	VerifyOtp(ctx context.Context, userId int64, input *model.MstockVerifyOtpInput) (*model.ResponseWrapper, error)
	PlaceFnOrder(ctx context.Context, userId int64, input *model.MstockOrderRequest) (*model.ResponseWrapper, error)
	GetProfile(ctx context.Context, userId int64) (*model.ResponseWrapper, error)
	RefreshAccessToken(ctx context.Context, userId int64) (*model.ResponseWrapper, error)
	Logout(ctx context.Context, userId int64) (*model.ResponseWrapper, error)
}

type MstockServiceImpl struct {
	angelOneWebSvc AngelOneWebSocket
	userSvc        UserService
}

func NewMstockService(angelOneWebSvc AngelOneWebSocket, userSvc UserService) MstockService {
	return &MstockServiceImpl{
		angelOneWebSvc: angelOneWebSvc,
		userSvc:        userSvc,
	}
}

func (s *MstockServiceImpl) Login(ctx context.Context, userId int64, input *model.MstockLoginInput) (*model.ResponseWrapper, error) {
	_, ok := s.getClient(userId)
	if ok {
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: true,
				Message: "Login successful",
				Data:    "S001",
			},
		}, nil
	}

	userIdStr := strconv.FormatInt(userId, 10)
	_, ok = cache.PendingLoginCache.Get(userIdStr)
	if ok {
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: false,
				Message: "OTP already sent",
				Data:    "E001",
			},
		}, nil
	}

	fnoClient := client.NewMstockClient(userId, input.APIKey, input.Username)
	res, err := fnoClient.Login(input)
	if err != nil {
		return nil, huma.Error500InternalServerError("E003")
	}

	cache.PendingLoginCache.Set(userIdStr, fnoClient, 2*time.Minute)
	cache.PendingLoginCache.Set("mstock:"+userIdStr, input, 5*time.Minute)
	return res, nil
}

func (s *MstockServiceImpl) VerifyOtp(ctx context.Context, userId int64, input *model.MstockVerifyOtpInput) (*model.ResponseWrapper, error) {
	userIdStr := strconv.FormatInt(userId, 10)
	val, exists := cache.PendingLoginCache.Get(userIdStr)
	if !exists {
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: false,
				Message: "OTP not sent",
				Data:    "E001",
			},
		}, nil
	}

	val2, exists := cache.PendingLoginCache.Get("mstock:" + userIdStr)
	if !exists {
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: false,
				Message: "OTP not sent",
				Data:    "E001",
			},
		}, nil
	}

	loginInput := val2.(*model.MstockLoginInput)
	fnoClient := val.(*client.MstockClient)
	res, err := fnoClient.Verify(input)

	if err != nil {
		return nil, huma.Error500InternalServerError("E003")
	}

	if res.Body.Success {
		cache.PendingLoginCache.Delete(userIdStr)
		cache.PendingLoginCache.Delete("mstock:" + userIdStr)
		cache.MstockClientCache.Set(userIdStr, fnoClient, util.GetDurationToMidnightIST())
		cache.GoSet("mstock:"+userIdStr, model.MstockRedisCache{
			AccessToken: fnoClient.AccessToken,
			Username:    fnoClient.MstockUserName,
			APIKey:      fnoClient.ApiKey,
		}, util.GetDurationToMidnightIST())

		user, err := s.userSvc.FindUser(ctx, 0, "", userId)
		if err != nil {
			log.Error().Err(err).Msg("Error getting user")
		} else if user.MstockConfig == (model.MstockConfig{}) {

			user.MstockConfig = model.MstockConfig{
				ApiKey:   fnoClient.ApiKey,
				Username: fnoClient.MstockUserName,
				Password: loginInput.Password,
			}

			err = s.userSvc.PatchUserData(ctx, userId, *user)
			if err != nil {
				log.Error().Err(err).Msg("Error updating user")
			} else {
				log.Info().Msg("User updated successfully")
			}
		}
	} else if res.Body.Data == "E002" {
		cache.PendingLoginCache.Delete(userIdStr)
		cache.PendingLoginCache.Delete("mstock:" + userIdStr)
	}

	return res, nil
}

func (s *MstockServiceImpl) PlaceFnOrder(ctx context.Context, userId int64, input *model.MstockOrderRequest) (*model.ResponseWrapper, error) {
	client, ok := s.getClient(userId)
	if !ok {
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: false,
				Message: "User not logged in",
				Data:    "E001",
			},
		}, nil
	}

	action := "CE"
	if input.Action == "PUT" {
		action = "PE"
	}

	key := input.Name + input.Expiry + input.Strike + action
	symbol, ok := cache.OptionCache.Get(key)
	if !ok {
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: false,
				Message: "Symbol not found",
				Data:    "E004",
			},
		}, nil
	}

	option := symbol.(model.OptionChain)
	qty, err := strconv.Atoi(option.LotSize)
	if err != nil {
		return nil, huma.Error500InternalServerError("E005")
	}

	request := &model.MstockOrderInput{
		Symbol:   option.MstockSymbol,
		Exchange: option.ExchangeType,
		Side:     "BUY",
		Type:     "MARKET",
		Qty:      strconv.Itoa(qty * input.Lots),
		Product:  "NRML",
		Validity: "DAY",
		Price:    "0",
	}

	resp, err := client.PlaceOrder(request)
	if err != nil {
		return nil, huma.Error500InternalServerError("E003")
	}

	if resp.Body.Success {
		orderId := resp.Body.Data.(string)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		go s.monitorAndSell(orderId, request, &option, input, client, ctx, cancel)
	}

	return resp, nil
}

func (s *MstockServiceImpl) GetProfile(ctx context.Context, userId int64) (*model.ResponseWrapper, error) {
	_, ok := s.getClient(userId)
	if !ok {

		user, err := s.userSvc.FindUser(ctx, 0, "", userId)
		if err != nil {
			return nil, huma.Error500InternalServerError("Error getting user")
		}

		if user.MstockConfig.ApiKey == "" || user.MstockConfig.Password == "" || user.MstockConfig.Username == "" {
			return &model.ResponseWrapper{
				Body: model.Response{
					Success: false,
					Message: "User not logged in",
					Data:    "E001",
				},
			}, nil
		}

		return &model.ResponseWrapper{
			Body: model.Response{
				Success: false,
				Message: "User not logged in",
				Data:    "E002",
			},
		}, nil
	}

	return &model.ResponseWrapper{
		Body: model.Response{
			Success: true,
			Message: "User logged in",
			Data:    userId,
		},
	}, nil
}

func (s *MstockServiceImpl) getClient(userId int64) (*client.MstockClient, bool) {
	key := strconv.FormatInt(userId, 10)
	val, ok := cache.MstockClientCache.Get(key)
	if ok {
		return val.(*client.MstockClient), true
	}

	var mstockCache model.MstockRedisCache
	database.RedisHelper.GetAsStruct("mstock:"+key, &mstockCache)
	if mstockCache.AccessToken != "" {
		fnoClient := client.NewMstockClient(userId, mstockCache.APIKey, mstockCache.Username)
		fnoClient.SetAccessToken(mstockCache.AccessToken)
		cache.MstockClientCache.Set(key, fnoClient, util.GetDurationToMidnightIST())
		return fnoClient, true
	}

	return nil, false
}

func (s *MstockServiceImpl) RefreshAccessToken(ctx context.Context, userId int64) (*model.ResponseWrapper, error) {
	user, err := s.userSvc.FindUser(ctx, 0, "", userId)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error getting user")
	}

	return s.Login(ctx, userId, &model.MstockLoginInput{
		APIKey:   user.MstockConfig.ApiKey,
		Password: user.MstockConfig.Password,
		Username: user.MstockConfig.Username,
	})

}

func (s *MstockServiceImpl) monitorAndSell(orderId string, input *model.MstockOrderInput,
	option *model.OptionChain, tradeRequest *model.MstockOrderRequest, client *client.MstockClient,
	ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	time.Sleep(1 * time.Second)

	var detail *model.MstockOrderResponse
	var err error

	for range 3 {
		detail, err = client.GetOrderDetails(orderId)
		if err == nil && detail != nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil || detail == nil {
		log.Error().Err(err).
			Str("orderId", orderId).
			Msg("Could not fetch order details after retries")
		return
	}

	avgPrice := detail.AveragePrice
	if avgPrice == 0 {
		return
	}

	exType := model.NFO
	if option.ExchangeType == "BFO" {
		exType = model.BFO
	}

	ch, err := s.angelOneWebSvc.Subscribe(option.AngelOneToken, exType)
	if err != nil {
		return
	}

	targetPrice := util.FixToTickOptions(avgPrice + tradeRequest.Profit)

	for {
		select {
		case <-ctx.Done():
			return
		case ltp, ok := <-ch:
			if !ok {
				return
			}
			if ltp >= targetPrice {
				client.PlaceOrder(&model.MstockOrderInput{
					Symbol:   input.Symbol,
					Exchange: input.Exchange,
					Side:     "SELL",
					Type:     "LIMIT",
					Qty:      input.Qty,
					Product:  input.Product,
					Validity: input.Validity,
					Price:    strconv.FormatFloat(targetPrice, 'f', 2, 64),
				})
				return
			}
		}
	}

}

func (s *MstockServiceImpl) Logout(ctx context.Context, userId int64) (*model.ResponseWrapper, error) {
	key := strconv.FormatInt(userId, 10)
	cache.MstockClientCache.Delete(key)
	cache.GoDelete("mstock:" + key)
	return &model.ResponseWrapper{
		Body: model.Response{
			Success: true,
			Message: "User logged out",
			Data:    userId,
		},
	}, nil
}
