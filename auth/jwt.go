package auth

import (
	"backend/model"
	"fmt"
	"strconv"
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
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   strconv.FormatInt(user.UserID, 10),
			Issuer:    "shahbaz-trades",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(SecretKey)
}

func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return SecretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	if claims.Issuer != "shahbaz-trades" {
		return nil, fmt.Errorf("invalid issuer")
	}

	expectedSub := strconv.FormatInt(claims.User.UserID, 10)
	if claims.Subject != expectedSub {
		return nil, fmt.Errorf("subject mismatch: identity integrity compromised")
	}

	return claims, nil
}
