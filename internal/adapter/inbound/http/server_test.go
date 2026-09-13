package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/7-solutions/backend-challenge/internal/adapter/outbound/jwt"
	"github.com/7-solutions/backend-challenge/internal/application/services/user"
)

const wiringTestSecret = "test-secret"

// wiringFakeService implements user.Service for the adapter wiring tests. It
// replaces the real service so the ServerInterface layer (apiHandler) can be
// driven end to end through the generated routes without a database.
type wiringFakeService struct {
	output user.UserOutput
	list   []user.UserOutput
	token  string
	err    error
}

func (f *wiringFakeService) Register(_ context.Context, _ user.CreateUserInput) (user.UserOutput, error) {
	return f.output, f.err
}

func (f *wiringFakeService) Login(_ context.Context, _ user.LoginInput) (user.LoginOutput, error) {
	return user.LoginOutput{Token: f.token, User: f.output}, f.err
}

func (f *wiringFakeService) GetByID(_ context.Context, _ string) (user.UserOutput, error) {
	return f.output, f.err
}

func (f *wiringFakeService) List(_ context.Context) ([]user.UserOutput, error) {
	return f.list, f.err
}

func (f *wiringFakeService) Update(_ context.Context, _ string, _ user.UpdateUserInput) (user.UserOutput, error) {
	return f.output, f.err
}

func (f *wiringFakeService) Delete(_ context.Context, _ string) error { return f.err }

func wiringIssueToken(t *testing.T) string {
	t.Helper()
	token, err := jwt.NewTokenManager(wiringTestSecret).Generate("u1")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func wiringApp(t *testing.T, svc user.Service) *fiber.App {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(svc, jwt.NewTokenManager(wiringTestSecret), logger)
}

func wiringDo(t *testing.T, app *fiber.App, method, path, body, token string) int {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func wiringSampleOutput() user.UserOutput {
	return user.UserOutput{ID: "u1", Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now().UTC()}
}

// TestServerWiring_Routes drives every route registered through the generated
// ServerInterface (apiHandler), verifying the adapter wiring delegates to the
// feature handlers for both public and authenticated endpoints.
func TestServerWiring_Routes(t *testing.T) {
	svc := &wiringFakeService{
		output: wiringSampleOutput(),
		list:   []user.UserOutput{wiringSampleOutput()},
		token:  "jwt-token",
	}
	app := wiringApp(t, svc)
	token := wiringIssueToken(t)

	protected := map[string]bool{
		"GET /api/v1/users":       true,
		"GET /api/v1/users/u1":    true,
		"PUT /api/v1/users/u1":    true,
		"DELETE /api/v1/users/u1": true,
	}

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"health check", "GET", "/health", "", fiber.StatusOK},
		{"register user", "POST", "/api/v1/users", `{"name":"Alice","email":"alice@example.com","password":"secret123"}`, fiber.StatusCreated},
		{"login", "POST", "/api/v1/auth/login", `{"email":"alice@example.com","password":"secret123"}`, fiber.StatusOK},
		{"list users", "GET", "/api/v1/users", "", fiber.StatusOK},
		{"get user", "GET", "/api/v1/users/u1", "", fiber.StatusOK},
		{"update user", "PUT", "/api/v1/users/u1", `{"name":"Alice Cooper"}`, fiber.StatusOK},
		{"delete user", "DELETE", "/api/v1/users/u1", "", fiber.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := ""
			if protected[tt.method+" "+tt.path] {
				auth = token
			}
			if status := wiringDo(t, app, tt.method, tt.path, tt.body, auth); status != tt.want {
				t.Fatalf("%s %s status = %d, want %d", tt.method, tt.path, status, tt.want)
			}
		})
	}
}

// TestServerWiring_ServiceError exercises the Server.WriteServiceError
// delegation: when a feature handler maps a service error, the assembling
// adapter's own responder is used end to end.
func TestServerWiring_ServiceError(t *testing.T) {
	app := wiringApp(t, &wiringFakeService{err: errors.New("boom")})

	if status := wiringDo(t, app, "POST", "/api/v1/users",
		`{"name":"Alice","email":"alice@example.com","password":"secret123"}`, ""); status != fiber.StatusInternalServerError {
		t.Fatalf("register with service error status = %d, want 500", status)
	}
}

// TestServerWiring_ProtectedRoutesRequireToken asserts the guarded handlers
// short-circuit in the apiHandler before reaching the feature handler when no
// bearer token is present.
func TestServerWiring_ProtectedRoutesRequireToken(t *testing.T) {
	app := wiringApp(t, &wiringFakeService{output: wiringSampleOutput()})

	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/users"},
		{"GET", "/api/v1/users/u1"},
		{"PUT", "/api/v1/users/u1"},
		{"DELETE", "/api/v1/users/u1"},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			if status := wiringDo(t, app, tt.method, tt.path, `{}`, ""); status != fiber.StatusUnauthorized {
				t.Fatalf("%s %s status = %d, want %d", tt.method, tt.path, status, fiber.StatusUnauthorized)
			}
		})
	}
}
