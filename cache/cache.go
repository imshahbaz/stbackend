package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var PendingUserCache = cache.New(5*time.Minute, 10*time.Minute)
var StrategyCache = cache.New(cache.NoExpiration, 0)
var MarginCache = cache.New(cache.NoExpiration, 0)
var ChartInkResponseCache = cache.New(1*time.Minute, 2*time.Minute)
var NseHistoryCache = cache.New(1*time.Hour, 10*time.Minute)
var UserAuthCache = cache.New(1*time.Hour, 10*time.Minute)
var HeatMapCache = cache.New(1*time.Hour, 10*time.Minute)
var OtpCache = cache.New(5*time.Minute, 10*time.Minute)
var RateLimiterCache = cache.New(10*time.Minute, 15*time.Minute)
