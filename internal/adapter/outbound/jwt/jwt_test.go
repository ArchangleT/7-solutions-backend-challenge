package jwt

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func newToken(t *testing.T, secret string, claims userClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		userID string
	}{
		{name: "signs subject", secret: "test-secret", userID: "u1"},
		{name: "signs empty subject", secret: "test-secret", userID: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTokenManager(tt.secret)
			token, err := tm.Generate(tt.userID)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if token == "" {
				t.Fatal("Generate() returned empty token")
			}
			if !strings.HasPrefix(token, "eyJ") {
				t.Errorf("Generate() token does not look like a JWT: %q", token)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	const secret = "test-secret"
	tm := NewTokenManager(secret)

	valid, err := tm.Generate("u1")
	if err != nil {
		t.Fatal(err)
	}

	expired := newToken(t, secret, userClaims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "u1",
		Issuer:    issuer,
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
	}})

	wrongIssuer := newToken(t, secret, userClaims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "u1",
		Issuer:    "other-service",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}})

	missingSubject := newToken(t, secret, userClaims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer:    issuer,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}})

	tampered := valid[:len(valid)-2] + "xx"

	hs384 := newTokenHS384(t, secret)

	otherSecret, err := NewTokenManager("other-secret").Generate("u1")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		secret  string
		token   string
		wantSub string
		wantErr bool
	}{
		{name: "valid token", secret: secret, token: valid, wantSub: "u1"},
		{name: "empty token", secret: secret, token: "", wantErr: true},
		{name: "garbage token", secret: secret, token: "not.a.jwt", wantErr: true},
		{name: "tampered signature", secret: secret, token: tampered, wantErr: true},
		{name: "expired token", secret: secret, token: expired, wantErr: true},
		{name: "wrong issuer", secret: secret, token: wrongIssuer, wantErr: true},
		{name: "missing subject", secret: secret, token: missingSubject, wantErr: true},
		{name: "wrong algorithm", secret: secret, token: hs384, wantErr: true},
		{name: "different secret", secret: secret, token: otherSecret, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTokenManager(tt.secret).Validate(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Validate() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
			if got != tt.wantSub {
				t.Errorf("Validate() = %q, want %q", got, tt.wantSub)
			}
		})
	}
}

func newTokenHS384(t *testing.T, secret string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS384, userClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u1",
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}
