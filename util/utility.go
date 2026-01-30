package util

import (
	"math"
	"math/rand/v2"
	"strconv"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func GenerateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}

func FixToTick(price float64) float64 {
	var tick float64

	switch {
	case price < 250:
		tick = 0.01
	case price < 1000:
		tick = 0.05
	case price < 5000:
		tick = 0.10
	case price < 10000:
		tick = 0.50
	default:
		tick = 1.00
	}

	return math.Round(price/tick) * tick
}

func FixToTickOptions(price float64) float64 {
	tick := 0.05
	return math.Round(price/tick) * tick
}

func NormalizeStrike(strikeStr string) string {
	if strikeStr == "" {
		return ""
	}

	f, err := strconv.ParseFloat(strikeStr, 64)
	if err != nil {
		return ""
	}

	if f > 1000000 {
		f = f / 100
	}

	return strconv.FormatFloat(math.Round(f), 'f', -1, 64)
}
