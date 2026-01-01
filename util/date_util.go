package util

import (
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

var (
	inputLayout  = "02-Jan-2006"
	outputLayout = "2006-01-02"
)

func ParseNseDate(nseDate string) (string, error) {
	cleanInput := strings.TrimSpace(nseDate)
	t, err := time.Parse(inputLayout, cleanInput)
	if err != nil {
		return "", err
	}
	return t.Format(outputLayout), nil
}

func NseCacheExpiryTime() time.Duration {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return cache.DefaultExpiration
	}

	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, loc)
	end := time.Date(now.Year(), now.Month(), now.Day(), 17, 30, 0, 0, loc)

	if now.After(start) && now.Before(end) {
		return 10 * time.Minute
	}

	return cache.DefaultExpiration
}

func ChartInkCacheExpiryTime() time.Duration {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return 10 * time.Minute
	}

	now := time.Now().In(loc)

	target := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, loc)
	to := time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, loc)

	if now.After(to) {
		return target.AddDate(0, 0, 1).Sub(now)
	} else if now.Before(target) {
		return target.Sub(now)
	}

	return 10 * time.Minute
}
