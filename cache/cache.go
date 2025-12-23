package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var PendingUserCache = cache.New(5*time.Minute, 10*time.Minute)
