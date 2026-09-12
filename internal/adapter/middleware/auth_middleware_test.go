package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"

	portsmocks "github.com/7-solutions/backend-challenge/internal/mocks/ports"
)

func TestAuthenticate(t *testing.T) {
	ctrl := gomock.NewController(t)
	tok := portsmocks.NewMockTokenManager(ctrl)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantErr    error
		wantUserID string
	}{
		{"missing header", "", fiber.StatusUnauthorized, ErrUnauthorized, ""},
		{"empty bearer value", "Bearer ", fiber.StatusUnauthorized, ErrUnauthorized, ""},
		{"wrong scheme", "Basic dXNlcjpwYXNz", fiber.StatusUnauthorized, ErrUnauthorized, ""},
		{"invalid token", "Bearer not-a-jwt", fiber.StatusUnauthorized, ErrUnauthorized, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "invalid token" {
				tok.EXPECT().Validate("not-a-jwt").Return("", errors.New("bad token"))
			}

			app := fiber.New()
			app.Get("/", func(c *fiber.Ctx) error {
				if err := Authenticate(c, tok); err != nil {
					return nil
				}
				return c.SendStatus(fiber.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestAuthenticate_ValidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	tok := portsmocks.NewMockTokenManager(ctrl)
	tok.EXPECT().Validate("good-token").Return("u1", nil)

	app := fiber.New()
	var gotUserID any
	app.Get("/", func(c *fiber.Ctx) error {
		if err := Authenticate(c, tok); err != nil {
			return nil
		}
		gotUserID = c.Locals(UserIDKey)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if gotUserID != "u1" {
		t.Errorf("locals[user_id] = %v, want u1", gotUserID)
	}
}

func TestAuthMiddleware(t *testing.T) {
	t.Run("passes through on valid token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		tok := portsmocks.NewMockTokenManager(ctrl)
		tok.EXPECT().Validate("good-token").Return("u1", nil)

		app := fiber.New()
		app.Get("/", AuthMiddleware(tok), func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer good-token")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
		}
	})

	t.Run("stops chain on invalid token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		tok := portsmocks.NewMockTokenManager(ctrl)
		tok.EXPECT().Validate("bad").Return("", errors.New("bad token"))

		app := fiber.New()
		called := false
		app.Get("/", AuthMiddleware(tok), func(c *fiber.Ctx) error {
			called = true
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer bad")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusUnauthorized)
		}
		if called {
			t.Error("downstream handler ran despite invalid token")
		}
	})
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"valid bearer", "Bearer abc.def", "abc.def"},
		{"case insensitive", "bearer xyz", "xyz"},
		{"extra spaces", "  Bearer   spaced-token  ", "spaced-token"},
		{"no scheme", "abc", ""},
		{"wrong scheme", "Basic abc", ""},
		{"empty", "", ""},
		{"bearer only", "Bearer", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bearerToken(tt.header); got != tt.want {
				t.Errorf("bearerToken(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}
