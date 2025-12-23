package auth

import (
	"backend/model"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte("your_super_secret_key_change_this")

type Claims struct {
	User model.UserDto `json:"user"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT with a specific email and issued-at time
func GenerateToken(user model.UserDto) (string, error) {
	now := time.Now()
	// 30 minutes for sliding window
	expirationTime := now.Add(30 * time.Minute)

	claims := &Claims{
		User: user,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now), // Critical for sliding expiry math
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ValidateToken now returns *Claims so the middleware can check expiration timing
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}
