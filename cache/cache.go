package cache

import (
	"backend/database"
	"backend/model"
	"backend/util"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

var EnableRedisCache = true

var pendingUserCache = cache.New(5*time.Minute, 10*time.Minute)
var StrategyCache = cache.New(cache.NoExpiration, 0)
var MarginCache = cache.New(cache.NoExpiration, 0)
var chartInkResponseCache = cache.New(1*time.Minute, 2*time.Minute)
var NseHistoryCache = cache.New(1*time.Hour, 10*time.Minute)
var UserAuthCache = cache.New(1*time.Hour, 10*time.Minute)
var HeatMapCache = cache.New(1*time.Hour, 10*time.Minute)
var OtpCache = cache.New(5*time.Minute, 10*time.Minute)
var RateLimiterCache = cache.New(10*time.Minute, 15*time.Minute)
var priceActionCache = cache.New(cache.NoExpiration, 0)
var YahooHistoryCache = cache.New(1*time.Hour, 10*time.Minute)

func getKeyAndExpiry(reqId string, cacheType model.UserCacheType) (string, time.Duration) {
	stringType := string(cacheType) + "_"
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

func GetUserCache(reqId string, userDto *model.UserDto, cacheType model.UserCacheType) (bool, error) {
	key, _ := getKeyAndExpiry(reqId, cacheType)
	if EnableRedisCache {
		return database.RedisHelper.GetAsStruct(key, userDto)
	}

	if value, ok := pendingUserCache.Get(key); ok {
		dto := value.(model.UserDto)
		*userDto = dto
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

//ChartInkResponseCache

func SetChartInkResponseCache(key string, value []model.StockMarginDto) {
	key = "chartink_result_" + key
	if EnableRedisCache {
		database.RedisHelper.Set(key, value, util.ChartInkCacheExpiryTime())
	} else {
		chartInkResponseCache.Set(key, value, util.ChartInkCacheExpiryTime())
	}
}

func GetChartInkResponseCache(key string, value *[]model.StockMarginDto) (bool, error) {
	key = "chartink_result_" + key
	if EnableRedisCache {
		return database.RedisHelper.GetAsStruct(key, value)
	}
	val, ok := chartInkResponseCache.Get(key)
	if ok {
		dto := val.([]model.StockMarginDto)
		*value = dto
		return ok, nil
	}
	return false, fmt.Errorf("Data not found")
}

//PriceActionCache

func SetPriceActionResponseCache(key string, value []model.ObResponse) {
	key = "price_action_result_" + key
	if EnableRedisCache {
		database.RedisHelper.Set(key, value, util.ChartInkCacheExpiryTime())
	} else {
		priceActionCache.Set(key, value, util.ChartInkCacheExpiryTime())
	}
}

func GetPriceActionResponseCache(key string, value *[]model.ObResponse) (bool, error) {
	key = "price_action_result_" + key
	if EnableRedisCache {
		return database.RedisHelper.GetAsStruct(key, value)
	}
	val, ok := chartInkResponseCache.Get(key)
	if ok {
		dto := val.([]model.ObResponse)
		*value = dto
		return ok, nil
	}
	return false, fmt.Errorf("Data not found")
}

func DeletePriceActionResponseCache(key string) {
	key = "price_action_result_" + key
	if EnableRedisCache {
		database.RedisHelper.Delete(key)
	}
	priceActionCache.Delete(key)
}
