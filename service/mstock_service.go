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
)

type MstockService interface {
	Login(ctx context.Context, input *model.MstockLoginInput) (*model.ResponseWrapper, error)
	VerifyOtp(ctx context.Context, input *model.MstockVerifyOtpInput) (*model.ResponseWrapper, error)
	PlaceFnOrder(ctx context.Context, input *model.MstockOrderInput) (*model.ResponseWrapper, error)
}

type MstockServiceImpl struct {
}

func NewMstockService() MstockService {
	return &MstockServiceImpl{}
}

func (s *MstockServiceImpl) Login(ctx context.Context, input *model.MstockLoginInput) (*model.ResponseWrapper, error) {
	_, ok := s.getClient(1)
	if ok {
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: true,
				Message: "Login successful",
				Data:    "S001",
			},
		}, nil
	}

	_, ok = cache.PendingLoginCache.Get(input.Username)
	if ok {
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: false,
				Message: "OTP already sent",
				Data:    "E001",
			},
		}, nil
	}

	fnoClient := client.NewMstockClient(1, input.APIKey, input.Username)
	cache.PendingLoginCache.Set("1", fnoClient, 5*time.Minute)
	res, err := fnoClient.Login(input)
	if err != nil {
		return nil, huma.Error500InternalServerError("E003")
	}
	return res, nil
}

func (s *MstockServiceImpl) VerifyOtp(ctx context.Context, input *model.MstockVerifyOtpInput) (*model.ResponseWrapper, error) {
	val, exists := cache.PendingLoginCache.Get("1")
	if !exists {
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: false,
				Message: "OTP not sent",
				Data:    "E001",
			},
		}, nil
	}

	fnoClient := val.(*client.MstockClient)
	res, err := fnoClient.Verify(input)

	if err != nil {
		return nil, huma.Error500InternalServerError("E003")
	}

	if res.Body.Success {
		cache.PendingLoginCache.Delete("1")
		cache.MstockClientCache.Set("1", fnoClient, util.GetDurationToMidnightIST())
		cache.GoSet("mstock:1", model.MstockRedisCache{
			AccessToken: fnoClient.AccessToken,
			Username:    fnoClient.MstockUserName,
			APIKey:      fnoClient.ApiKey,
		}, util.GetDurationToMidnightIST())
	}

	return res, nil
}

func (s *MstockServiceImpl) PlaceFnOrder(ctx context.Context, input *model.MstockOrderInput) (*model.ResponseWrapper, error) {
	client, ok := s.getClient(1)
	if !ok {
		return &model.ResponseWrapper{
			Body: model.Response{
				Success: false,
				Message: "User not logged in",
				Data:    "E001",
			},
		}, nil
	}

	resp, err := client.PlaceOrder(input)
	if err != nil {
		return nil, huma.Error500InternalServerError("E003")
	}

	return resp, nil
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
