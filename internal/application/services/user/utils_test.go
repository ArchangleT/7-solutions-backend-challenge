package user

import (
	"testing"
	"time"

	domain "github.com/7-solutions/backend-challenge/internal/domain/user"
)

func TestToOutput(t *testing.T) {
	created := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	u := &domain.User{ID: "u1", Name: "Alice", Email: "alice@example.com", Password: "hashed", CreatedAt: created}

	got := toOutput(u)
	want := UserOutput{ID: "u1", Name: "Alice", Email: "alice@example.com", CreatedAt: created}

	if got != want {
		t.Errorf("toOutput() = %+v, want %+v", got, want)
	}
}

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "lowercase unchanged", input: "alice@example.com", want: "alice@example.com"},
		{name: "trim surrounding spaces", input: "  alice@example.com  ", want: "alice@example.com"},
		{name: "uppercase domain lowered", input: "Alice@Example.COM", want: "alice@example.com"},
		{name: "spaces and case combined", input: "  BoB@Example.COM ", want: "bob@example.com"},
		{name: "empty stays empty", input: "  ", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeEmail(tt.input); got != tt.want {
				t.Errorf("normalizeEmail(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "empty", input: "", want: false},
		{name: "space only", input: "   ", want: false},
		{name: "bare local part", input: "alice", want: false},
		{name: "missing domain", input: "alice@", want: false},
		{name: "missing local at", input: "@example.com", want: false},
		{name: "double at", input: "ali@ce@example.com", want: false},
		{name: "no tld is still parseable", input: "alice@example", want: true},
		{name: "with display name", input: "Alice <alice@example.com>", want: false},
		{name: "trailing whitespace", input: "alice@example.com ", want: false},
		{name: "valid lowercase", input: "alice@example.com", want: true},
		{name: "valid subdomain", input: "alice@mail.example.com", want: true},
		{name: "valid plus", input: "alice+tag@example.com", want: true},
		{name: "valid hyphen domain", input: "alice@my-example.com", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validEmail(tt.input); got != tt.want {
				t.Errorf("validEmail(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}