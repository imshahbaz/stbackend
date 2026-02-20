package util

import (
	"time"
)

const (
	MarketOpenHour           = 9
	MarketOpenMinute         = 15
	MarketCloseHour          = 15
	MarketCloseMinute        = 15
	MarketSquareOffHour      = 15
	MarketSquareOffMin       = 30
	MarketClosingGraceHour   = 15
	MarketClosingGraceMinute = 25
)

// IsMarketOpen checks if the current time is within market hours
func IsMarketOpen() bool {
	now := time.Now().In(IstLocation)

	// Start check
	if now.Hour() < MarketOpenHour || (now.Hour() == MarketOpenHour && now.Minute() < MarketOpenMinute) {
		return false
	}

	// Close check
	if now.Hour() > MarketCloseHour || (now.Hour() == MarketCloseHour && now.Minute() >= MarketCloseMinute) {
		return false
	}

	// Weekend check
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}

	return true
}

// IsMarketClosedForTrading checks if it's past the market close time (15:15)
func IsMarketClosedForTrading() bool {
	now := time.Now().In(IstLocation)
	if now.Hour() > MarketCloseHour {
		return true
	}
	if now.Hour() == MarketCloseHour && now.Minute() >= MarketCloseMinute {
		return true
	}
	return false
}

// IsSquareOffTimeReached checks if it's past the square-off time (15:30)
func IsSquareOffTimeReached() bool {
	now := time.Now().In(IstLocation)
	if now.Hour() > MarketSquareOffHour {
		return true
	}
	if now.Hour() == MarketSquareOffHour && now.Minute() >= MarketSquareOffMin {
		return true
	}
	return false
}

// IsPastClosingGrace checks if it's past the closing grace time (15:25)
func IsPastClosingGrace() bool {
	now := time.Now().In(IstLocation)
	if now.Hour() > MarketClosingGraceHour {
		return true
	}
	if now.Hour() == MarketClosingGraceHour && now.Minute() >= MarketClosingGraceMinute {
		return true
	}
	return false
}

// ShouldExitTradingLoop is a helper that returns true if market is closed or square-off reached
func ShouldExitTradingLoop() bool {
	return IsPastClosingGrace() || IsSquareOffTimeReached()
}
