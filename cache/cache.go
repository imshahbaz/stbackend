package cache

import (
	"backend/database"
	"backend/model"
	"backend/util"
	"time"

	"github.com/patrickmn/go-cache"
)

var StrategyCache = cache.New(cache.NoExpiration, 0)
var MarginCache = cache.New(cache.NoExpiration, 0)
var RateLimiterCache = cache.New(10*time.Minute, 15*time.Minute)

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
	database.RedisHelper.Set(key, userDto, expiry)
}

func GetUserCache(reqId string, userDto *model.UserDto, cacheType model.UserCacheType) (bool, error) {
	key, _ := getKeyAndExpiry(reqId, cacheType)
	return database.RedisHelper.GetAsStruct(key, userDto)
}

func DeleteUserCache(reqId string, cacheType model.UserCacheType) {
	key, _ := getKeyAndExpiry(reqId, cacheType)
	database.RedisHelper.Delete(key)
}

//ChartInkResponseCache

func SetChartInkResponseCache(key string, value []model.StockMarginDto) {
	key = "chartink_result_" + key
	database.RedisHelper.Set(key, value, util.NseCacheExpiryTime())
}

func GetChartInkResponseCache(key string, value *[]model.StockMarginDto) (bool, error) {
	key = "chartink_result_" + key
	return database.RedisHelper.GetAsStruct(key, value)
}

//PriceActionCache

func SetPriceActionResponseCache(key string, value []model.ObResponse) {
	key = "price_action_result_" + key
	database.RedisHelper.Set(key, value, util.NseCacheExpiryTime())
}

func GetPriceActionResponseCache(key string, value *[]model.ObResponse) (bool, error) {
	key = "price_action_result_" + key
	return database.RedisHelper.GetAsStruct(key, value)
}

func DeletePriceActionResponseCache(key string) {
	key = "price_action_result_" + key
	database.RedisHelper.Delete(key)
}
