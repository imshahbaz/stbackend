package util

import (
	"backend/model"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"google.golang.org/api/idtoken"
)

func SignState(uuid, key string) string {
	hmacKey := []byte(key)
	h := hmac.New(sha256.New, hmacKey)
	h.Write([]byte(uuid))
	return uuid + "." + hex.EncodeToString(h.Sum(nil))
}

func ExtractAndVerify(signedCode, key string) (string, bool) {
	parts := strings.Split(signedCode, ".")
	if len(parts) != 2 {
		return "", false
	}

	uuid := parts[0]
	sig, err := hex.DecodeString(parts[1])
	if err != nil {
		return "", false
	}

	hmacKey := []byte(key)
	h := hmac.New(sha256.New, hmacKey)
	h.Write([]byte(uuid))
	expectedSig := h.Sum(nil)

	if !hmac.Equal(sig, expectedSig) {
		return "", false
	}

	return uuid, true
}

func ValidateGoogleIDToken(ctx context.Context, idToken string, googleClientID string) (*model.GoogleUser, error) {

	if idToken == "" {
		return nil, errors.New("empty id token")
	}

	payload, err := idtoken.Validate(ctx, idToken, googleClientID)
	if err != nil {
		return nil, err
	}

	email, _ := payload.Claims["email"].(string)
	sub, _ := payload.Claims["sub"].(string)
	if email == "" || sub == "" {
		return nil, errors.New("invalid google token payload")
	}

	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	familyName, _ := payload.Claims["family_name"].(string)
	givenName, _ := payload.Claims["given_name"].(string)

	return &model.GoogleUser{
		Email:         email,
		VerifiedEmail: emailVerified,
		Name:          name,
		Picture:       picture,
		FamilyName:    familyName,
		GivenName:     givenName,
	}, nil
}
