package util

import (
	"math"
	"math/rand/v2"
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
