package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const otpLength = 6

// GenerateOtp creates a secure 6-digit OTP string
func GenerateOtp() (string, error) {
	// 1. Define the upper bound (10^6 = 1,000,000)
	// This means the random number will be between 0 and 999,999
	max := big.NewInt(1000000)

	// 2. Generate secure random number
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure random number: %w", err)
	}

	// 3. Format with leading zeros (equivalent to String.format("%06d", otp))
	return fmt.Sprintf("%0*d", otpLength, n.Int64()), nil
}
