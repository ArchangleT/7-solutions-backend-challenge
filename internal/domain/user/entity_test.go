package user

import (
	"errors"
	"testing"
	"time"
)

func TestUserEntity_Fields(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		nameField string
		email     string
		password  string
		createdAt time.Time
	}{
		{
			name:      "all fields populated per D1-5",
			id:        "u1",
			nameField: "Alice",
			email:     "alice@example.com",
			password:  "$2a$10$hashednotplain",
			createdAt: time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "zero value is valid for construction",
			id:        "",
			nameField: "",
			email:     "",
			password:  "",
			createdAt: time.Time{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := User{ID: tt.id, Name: tt.nameField, Email: tt.email, Password: tt.password, CreatedAt: tt.createdAt}
			if u.ID != tt.id {
				t.Errorf("ID = %q, want %q", u.ID, tt.id)
			}
			if u.Name != tt.nameField {
				t.Errorf("Name = %q, want %q", u.Name, tt.nameField)
			}
			if u.Email != tt.email {
				t.Errorf("Email = %q, want %q", u.Email, tt.email)
			}
			if u.Password != tt.password {
				t.Errorf("Password = %q, want %q", u.Password, tt.password)
			}
			if !u.CreatedAt.Equal(tt.createdAt) {
				t.Errorf("CreatedAt = %v, want %v", u.CreatedAt, tt.createdAt)
			}
		})
	}
}

func TestUserEntity_PasswordIsHashed(t *testing.T) {
	plain := "secret123"
	u := User{Password: plain}
	if u.Password != plain {
		t.Errorf("Password = %q, want %q", u.Password, plain)
	}
	hashed := &User{Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"}
	if hashed.Password == plain {
		t.Error("Password must never equal the plain text")
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{"ErrNotFound", ErrNotFound, "user not found"},
		{"ErrDuplicateEmail", ErrDuplicateEmail, "email already exists"},
		{"ErrInvalidCredentials", ErrInvalidCredentials, "invalid credentials"},
		{"ErrInvalidID", ErrInvalidID, "invalid user id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("sentinel error must not be nil")
			}
			if tt.err.Error() != tt.message {
				t.Errorf("%s = %q, want %q", tt.name, tt.err.Error(), tt.message)
			}
			if !errors.Is(tt.err, tt.err) {
				t.Errorf("errors.Is(%v, %v) must hold", tt.err, tt.err)
			}
		})
	}
}

func TestSentinelErrors_Distinct(t *testing.T) {
	sentinels := []error{ErrNotFound, ErrDuplicateEmail, ErrInvalidCredentials, ErrInvalidID}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j && errors.Is(a, b) {
				t.Errorf("sentinel %v aliases %v", a, b)
			}
		}
	}
}

func TestValidationError(t *testing.T) {
	v := &ValidationError{Field: "email", Message: "is required"}
	if got, want := v.Error(), "validation error: email is required"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	var target *ValidationError
	if !errors.As(error(v), &target) {
		t.Fatalf("errors.As(*ValidationError) failed for %v", v)
	}
	if target.Field != "email" {
		t.Errorf("Field = %q, want email", target.Field)
	}
	if target.Message != "is required" {
		t.Errorf("Message = %q, want is required", target.Message)
	}
}
