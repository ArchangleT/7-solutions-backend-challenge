// Package jwt implements the TokenManager port using HMAC-signed JWTs (HS256).
package jwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const (
	issuer   = "user-management-api"
	tokenTTL = 24 * time.Hour
)

// TokenManager satisfies ports.TokenManager.
type TokenManager struct {
	secret []byte
}

func NewTokenManager(secret string) *TokenManager {
	return &TokenManager{secret: []byte(secret)}
}

type userClaims struct {
	jwtlib.RegisteredClaims
}

// Generate issues a signed HS256 token carrying the user ID as subject.
func (t *TokenManager) Generate(userID string) (string, error) {
	now := time.Now()
	claims := userClaims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID,
			Issuer:    issuer,
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(tokenTTL)),
		},
	}
	signed, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Validate verifies the token signature, issuer and validity period, and
// returns the user ID stored in the subject claim.
func (t *TokenManager) Validate(token string) (string, error) {
	parsed, err := jwtlib.ParseWithClaims(token, &userClaims{}, func(pt *jwtlib.Token) (any, error) {
		if _, ok := pt.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", pt.Header["alg"])
		}
		return t.secret, nil
	}, jwtlib.WithIssuer(issuer), jwtlib.WithValidMethods([]string{jwtlib.SigningMethodHS256.Alg()}), jwtlib.WithExpirationRequired())
	if err != nil {
		return "", err
	}

	claims, ok := parsed.Claims.(*userClaims)
	if !ok || !parsed.Valid || strings.TrimSpace(claims.Subject) == "" {
		return "", errors.New("invalid token claims")
	}
	return claims.Subject, nil
}
