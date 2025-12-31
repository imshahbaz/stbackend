package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const otpLength = 6

func GenerateOtp() (string, error) {
	max := big.NewInt(1000000)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure random number: %w", err)
	}

	return fmt.Sprintf("%0*d", otpLength, n.Int64()), nil
}
