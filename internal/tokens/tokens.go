package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// --- Magic link token (JWT, short-lived) ---

type MagicLinkClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateMagicToken(email, secret string) (string, error) {
	claims := MagicLinkClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateMagicToken(tokenStr, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &MagicLinkClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*MagicLinkClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	return claims.Email, nil
}

// --- Device token (256-bit random, stored as SHA256 hash) ---

func GenerateDeviceToken() (token string, hash string, err error) {
	raw := make([]byte, 32) // 256 bits
	if _, err = rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	hash = HashToken(token)
	return token, hash, nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
