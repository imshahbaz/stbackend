package cache

import (
	"backend/database"
	"backend/model"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

var EnableRedisCache = true

var pendingUserCache = cache.New(5*time.Minute, 10*time.Minute)
var StrategyCache = cache.New(cache.NoExpiration, 0)
var MarginCache = cache.New(cache.NoExpiration, 0)
var ChartInkResponseCache = cache.New(1*time.Minute, 2*time.Minute)
var NseHistoryCache = cache.New(1*time.Hour, 10*time.Minute)
var UserAuthCache = cache.New(1*time.Hour, 10*time.Minute)
var HeatMapCache = cache.New(1*time.Hour, 10*time.Minute)
var OtpCache = cache.New(5*time.Minute, 10*time.Minute)
var RateLimiterCache = cache.New(10*time.Minute, 15*time.Minute)
var PriceActionCache = cache.New(cache.NoExpiration, 0)
var YahooHistoryCache = cache.New(1*time.Hour, 10*time.Minute)

func getKeyAndExpiry(reqId string, cacheType model.UserCacheType) (string, time.Duration) {
	stringType := string(cacheType)
	switch cacheType {
	case model.Truecaller:
		return stringType + reqId, 2 * time.Minute
	case model.Signup, model.CredUpdate:
		return stringType + reqId, 5 * time.Minute
	default:
		return "default", 10 * time.Second
	}
}

func SetUserCache(reqId string, userDto model.UserDto, cacheType model.UserCacheType) {
	key, expiry := getKeyAndExpiry(reqId, cacheType)
	if EnableRedisCache {
		database.RedisHelper.Set(key, userDto, expiry)
	} else {
		pendingUserCache.Set(key, userDto, expiry)
	}
}

func GetUserCache(reqId string, userDto model.UserDto, cacheType model.UserCacheType) (bool, error) {
	key, _ := getKeyAndExpiry(reqId, cacheType)
	if EnableRedisCache {
		return database.RedisHelper.GetAsStruct(key, userDto)
	}

	if value, ok := pendingUserCache.Get(key); ok {
		userDto = value.(model.UserDto)
		return ok, nil
	}
	return false, fmt.Errorf("Data not found")
}

func DeleteUserCache(reqId string, cacheType model.UserCacheType) {
	key, _ := getKeyAndExpiry(reqId, cacheType)
	if EnableRedisCache {
		database.RedisHelper.Delete(key)
	}
	pendingUserCache.Delete(key)
}
