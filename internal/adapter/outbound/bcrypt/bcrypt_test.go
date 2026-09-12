package bcrypt

import (
	"strings"
	"testing"
)

func TestHash(t *testing.T) {
	h := New()
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "valid password", password: "secret123"},
		{name: "empty password", password: ""},
		{name: "password longer than 72 bytes", password: strings.Repeat("a", 73), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashed, err := h.Hash(tt.password)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Hash() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Hash() error = %v, want nil", err)
			}
			if hashed == tt.password {
				t.Error("Hash() returned the plain-text password")
			}
		})
	}
}

func TestCompare(t *testing.T) {
	h := New()
	hashed, err := h.Hash("secret123")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		hashed string
		plain  string
		want   bool
	}{
		{name: "correct password", hashed: hashed, plain: "secret123", want: true},
		{name: "wrong password", hashed: hashed, plain: "wrong"},
		{name: "empty plain", hashed: hashed, plain: ""},
		{name: "malformed hash", hashed: "not-a-bcrypt-hash", plain: "secret123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := h.Compare(tt.hashed, tt.plain); got != tt.want {
				t.Errorf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashProducesDifferentSalt(t *testing.T) {
	h := New()

	a, err := h.Hash("secret123")
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.Hash("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Error("same password produced identical hashes; missing salt")
	}
}
