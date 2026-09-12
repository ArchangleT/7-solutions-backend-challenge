package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidate(t *testing.T) {
	tm := NewTokenManager("test-secret")

	token, err := tm.Generate("u1")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("Generate() returned empty token")
	}

	got, err := tm.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got != "u1" {
		t.Errorf("Validate() = %q, want u1", got)
	}
}

func TestValidate_RejectsInvalidToken(t *testing.T) {
	tm := NewTokenManager("test-secret")

	for name, token := range map[string]string{
		"empty":           "",
		"garbage":         "not.a.jwt",
		"wrong signature": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1MSJ9.some-tampered-signature",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := tm.Validate(token); err == nil {
				t.Error("Validate() should reject invalid token")
			}
		})
	}
}

func TestValidate_RejectsDifferentSecret(t *testing.T) {
	token, err := NewTokenManager("secret-a").Generate("u1")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewTokenManager("secret-b").Validate(token); err == nil {
		t.Error("Validate() should reject token signed with a different secret")
	}
}

func TestValidate_RejectsWrongAlg(t *testing.T) {
	// a token signed with HS384 (different algorithm) must be rejected
	tm := NewTokenManager("test-secret")
	claims := userClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "u1", Issuer: issuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tm.Validate(token); err == nil {
		t.Error("Validate() should reject HS384 token when only HS256 is allowed")
	}
}
