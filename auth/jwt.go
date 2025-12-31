package auth

import (
	"backend/model"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var SecretKey = []byte("")

type Claims struct {
	User model.UserDto `json:"user"`
	jwt.RegisteredClaims
}

func GenerateToken(user model.UserDto) (string, error) {
	now := time.Now()
	expirationTime := now.Add(30 * time.Minute)

	claims := &Claims{
		User: user,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now), // Critical for sliding expiry math
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(SecretKey)
}

func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}
