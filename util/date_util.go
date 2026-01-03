package util

import (
	"strings"
	"time"
)

var (
	InputLayout  = "02-Jan-2006"
	OutputLayout = "2006-01-02"
)

var (
	IstLocation     *time.Location
	secondsIn30Days = int64(30 * 24 * 60 * 60)
)

func init() {
	var err error
	IstLocation, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		IstLocation = time.FixedZone("IST", 5.5*60*60)
	}
}

func ParseNseDate(nseDate string) (string, error) {
	cleanInput := strings.TrimSpace(nseDate)
	t, err := time.Parse(InputLayout, cleanInput)
	if err != nil {
		return "", err
	}
	return t.Format(OutputLayout), nil
}

func NseCacheExpiryTime() time.Duration {
	now := time.Now().In(IstLocation)
	start := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, IstLocation)
	end := time.Date(now.Year(), now.Month(), now.Day(), 17, 30, 0, 0, IstLocation)

	if now.After(start) && now.Before(end) {
		return 10 * time.Minute
	}

	if now.Before(start) {
		return time.Until(start)
	}

	return time.Until(start.AddDate(0, 0, 1))
}
