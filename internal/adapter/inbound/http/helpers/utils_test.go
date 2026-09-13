package helpers

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	domain "github.com/ArchangleT/7-solutions-backend-challenge/internal/domain/user"
)

func newTestApp(t *testing.T, decode func(c *fiber.Ctx) (bool, error)) *fiber.App {
	t.Helper()
	// No BodyLimit here: the oversized-body path is exercised by DecodeBody
	// itself in TestDecodeBody_TooLarge, so Fiber must not reject it first.
	app := fiber.New()
	app.Post("/decode", func(c *fiber.Ctx) error {
		if ok, err := decode(c); err != nil || !ok {
			return err
		}
		_, _ = c.WriteString("ok")
		return nil
	})
	app.Post("/error", func(c *fiber.Ctx) error {
		return WriteServiceError(c, slog.New(slog.NewTextHandler(io.Discard, nil)), errors.New("boom"))
	})
	return app
}

func do(t *testing.T, app *fiber.App, body string) (*http.Response, []byte) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader(body))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	raw, _ := io.ReadAll(resp.Body)
	return resp, raw
}

func TestDecodeBody(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
		wantOK     bool
	}{
		{
			name:       "valid single object",
			body:       `{"name":"Alice","email":"alice@example.com"}`,
			wantStatus: fiber.StatusOK,
			wantBody:   "ok",
			wantOK:     true,
		},
		{
			name:       "malformed json",
			body:       `{"name":`,
			wantStatus: fiber.StatusBadRequest,
			wantBody:   `{"error":"invalid request body"}`,
			wantOK:     false,
		},
		{
			name:       "unknown field rejected",
			body:       `{"name":"Alice","bogus":1}`,
			wantStatus: fiber.StatusBadRequest,
			wantBody:   `{"error":"invalid request body"}`,
			wantOK:     false,
		},
		{
			name:       "trailing garbage after object",
			body:       `{"name":"Alice"} {}`,
			wantStatus: fiber.StatusBadRequest,
			wantBody:   `{"error":"request body must contain a single JSON object"}`,
			wantOK:     false,
		},
		{
			name:       "two adjacent objects",
			body:       `{"name":"Alice"}{"name":"Bob"}`,
			wantStatus: fiber.StatusBadRequest,
			wantBody:   `{"error":"request body must contain a single JSON object"}`,
			wantOK:     false,
		},
		{
			name:       "empty body",
			body:       ``,
			wantStatus: fiber.StatusBadRequest,
			wantBody:   `{"error":"invalid request body"}`,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t, func(c *fiber.Ctx) (bool, error) {
				var dst map[string]string
				if !DecodeBody(c, &dst) {
					return false, nil
				}
				return true, nil
			})
			resp, raw := do(t, app, tt.body)

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d (body %s)", resp.StatusCode, tt.wantStatus, raw)
			}
			if string(raw) != tt.wantBody {
				t.Errorf("body = %q, want %q", raw, tt.wantBody)
			}
		})
	}
}

func TestDecodeBody_TooLarge(t *testing.T) {
	app := newTestApp(t, func(c *fiber.Ctx) (bool, error) {
		var dst map[string]string
		if !DecodeBody(c, &dst) {
			return false, nil
		}
		return true, nil
	})

	big := `{"data": "` + strings.Repeat("a", MaxBodyBytes) + `"}`
	resp, raw := do(t, app, big)

	if resp.StatusCode != fiber.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusRequestEntityTooLarge)
	}
	if string(raw) != `{"error":"request body too large"}` {
		t.Errorf("body = %q, want request body too large", raw)
	}
}

func TestDecodeBody_DecodesIntoStruct(t *testing.T) {
	type userInput struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	app := fiber.New()
	var got userInput
	app.Post("/", func(c *fiber.Ctx) error {
		if !DecodeBody(c, &got) {
			return nil
		}
		return c.SendStatus(fiber.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Alice","email":"a@example.com"}`))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusAccepted)
	}
	if got.Name != "Alice" || got.Email != "a@example.com" {
		t.Errorf("decoded = %+v, want Alice/a@example.com", got)
	}
}

func TestWriteServiceError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "validation error → 400 with fields",
			err:        &domain.ValidationError{Field: "name", Message: "is required"},
			wantStatus: fiber.StatusBadRequest,
			wantBody:   `{"error":"validation error: name is required","fields":{"name":"is required"}}`,
		},
		{
			name:       "invalid id → 400",
			err:        domain.ErrInvalidID,
			wantStatus: fiber.StatusBadRequest,
			wantBody:   `{"error":"invalid user id"}`,
		},
		{
			name:       "not found → 404",
			err:        domain.ErrNotFound,
			wantStatus: fiber.StatusNotFound,
			wantBody:   `{"error":"user not found"}`,
		},
		{
			name:       "duplicate email → 409",
			err:        domain.ErrDuplicateEmail,
			wantStatus: fiber.StatusConflict,
			wantBody:   `{"error":"email already exists"}`,
		},
		{
			name:       "invalid credentials → 401",
			err:        domain.ErrInvalidCredentials,
			wantStatus: fiber.StatusUnauthorized,
			wantBody:   `{"error":"invalid credentials"}`,
		},
		{
			name:       "unhandled error → 500",
			err:        errors.New("boom"),
			wantStatus: fiber.StatusInternalServerError,
			wantBody:   `{"error":"internal server error"}`,
		},
		{
			name:       "wrapped sentinel still mapped",
			err:        errors.Join(domain.ErrNotFound, errors.New("driver detail")),
			wantStatus: fiber.StatusNotFound,
			wantBody:   `{"error":"user not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/", func(c *fiber.Ctx) error {
				return WriteServiceError(c, logger, tt.err)
			})

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(nil))
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if string(raw) != tt.wantBody {
				t.Errorf("body = %q, want %q", raw, tt.wantBody)
			}
		})
	}
}
