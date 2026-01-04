package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
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
