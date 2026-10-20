package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	httpapi "github.com/ArchangleT/7-solutions-backend-challenge/internal/adapter/inbound/http"
	"github.com/ArchangleT/7-solutions-backend-challenge/internal/adapter/middleware"
	"github.com/ArchangleT/7-solutions-backend-challenge/internal/adapter/outbound/jwt"
	"github.com/ArchangleT/7-solutions-backend-challenge/internal/application/services/user"
	domain "github.com/ArchangleT/7-solutions-backend-challenge/internal/domain/user"
)

// fakeService implements user.Service for handler-level tests.
type fakeService struct {
	output  user.UserOutput
	list    []user.UserOutput
	results []user.SearchResultOutput
	token   string
	err     error
}

func (f *fakeService) Register(_ context.Context, _ user.CreateUserInput) (user.UserOutput, error) {
	return f.output, f.err
}

func (f *fakeService) Login(_ context.Context, _ user.LoginInput) (user.LoginOutput, error) {
	return user.LoginOutput{Token: f.token, User: f.output}, f.err
}

func (f *fakeService) GetByID(_ context.Context, _ string) (user.UserOutput, error) {
	return f.output, f.err
}

func (f *fakeService) List(_ context.Context) ([]user.UserOutput, error) {
	return f.list, f.err
}

func (f *fakeService) Search(_ context.Context, _ user.SearchUsersInput) ([]user.SearchResultOutput, error) {
	return f.results, f.err
}

func (f *fakeService) Update(_ context.Context, _ string, _ user.UpdateUserInput) (user.UserOutput, error) {
	return f.output, f.err
}

func (f *fakeService) Delete(_ context.Context, _ string) error { return f.err }

func newTestApp(t *testing.T, svc user.Service) *fiber.App {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httpapi.NewServer(svc, jwt.NewTokenManager("test-secret"), logger)
}

func issueToken(t *testing.T) string {
	t.Helper()
	return issueTokenFor(t, "u1")
}

func issueTokenFor(t *testing.T, subject string) string {
	t.Helper()
	token, err := jwt.NewTokenManager("test-secret").Generate(subject)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func doRequest(t *testing.T, app *fiber.App, method, path, body, token string) (int, []byte) {
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
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func sampleOutput() user.UserOutput {
	return user.UserOutput{ID: "u1", Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now()}
}

func TestRegister_Created(t *testing.T) {
	svc := &fakeService{output: sampleOutput()}
	app := newTestApp(t, svc)

	status, body := doRequest(t, app, "POST", "/api/v1/users",
		`{"name":"Alice","email":"alice@example.com","password":"secret123"}`, "")

	if status != fiber.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", status, body)
	}
	var got user.UserOutput
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "u1" {
		t.Errorf("body = %+v", got)
	}
}

func TestRegister_BadRequest(t *testing.T) {
	app := newTestApp(t, &fakeService{})

	for name, body := range map[string]string{
		"malformed json":    `{"name":`,
		"unknown field":     `{"name":"A","email":"a@b.com","password":"secret123","role":"admin"}`,
		"not a json object": `[1,2,3]`,
	} {
		t.Run(name, func(t *testing.T) {
			status, _ := doRequest(t, app, "POST", "/api/v1/users", body, "")
			if status != fiber.StatusBadRequest {
				t.Fatalf("status = %d, want 400", status)
			}
		})
	}
}

func TestRegister_Conflict(t *testing.T) {
	svc := &fakeService{err: domain.ErrDuplicateEmail}
	app := newTestApp(t, svc)

	status, _ := doRequest(t, app, "POST", "/api/v1/users",
		`{"name":"Alice","email":"alice@example.com","password":"secret123"}`, "")
	if status != fiber.StatusConflict {
		t.Fatalf("status = %d, want 409", status)
	}
}

func TestRegister_ValidationError(t *testing.T) {
	svc := &fakeService{err: &domain.ValidationError{Field: "email", Message: "is invalid"}}
	app := newTestApp(t, svc)

	status, body := doRequest(t, app, "POST", "/api/v1/users",
		`{"name":"A","email":"bad","password":"secret123"}`, "")
	if status != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
	var errBody struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(body, &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody.Fields["email"] != "is invalid" {
		t.Errorf("fields = %v, want email field", errBody.Fields)
	}
}

func TestLogin_Success(t *testing.T) {
	svc := &fakeService{token: "jwt-token", output: sampleOutput()}
	app := newTestApp(t, svc)

	status, body := doRequest(t, app, "POST", "/api/v1/auth/login",
		`{"email":"alice@example.com","password":"secret123"}`, "")
	if status != fiber.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", status, body)
	}
	var out user.LoginOutput
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	if out.Token != "jwt-token" {
		t.Errorf("token = %q, want jwt-token", out.Token)
	}
}

func TestLogin_Unauthorized(t *testing.T) {
	svc := &fakeService{err: domain.ErrInvalidCredentials}
	app := newTestApp(t, svc)

	status, _ := doRequest(t, app, "POST", "/api/v1/auth/login",
		`{"email":"alice@example.com","password":"wrong"}`, "")
	if status != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestLogin_BadRequest(t *testing.T) {
	app := newTestApp(t, &fakeService{})

	for name, body := range map[string]string{
		"malformed json": `{"email":`,
		"unknown field":  `{"email":"a@b.com","password":"x","role":"admin"}`,
	} {
		t.Run(name, func(t *testing.T) {
			status, _ := doRequest(t, app, "POST", "/api/v1/auth/login", body, "")
			if status != fiber.StatusBadRequest {
				t.Fatalf("status = %d, want 400", status)
			}
		})
	}
}

func TestProtectedEndpoints_RequireAuth(t *testing.T) {
	app := newTestApp(t, &fakeService{output: sampleOutput()})
	token := issueToken(t)

	tests := []struct {
		method, path string
	}{
		{"GET", "/api/v1/users"},
		{"GET", "/api/v1/users/u1"},
		{"PUT", "/api/v1/users/u1"},
		{"DELETE", "/api/v1/users/u1"},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path+" missing token", func(t *testing.T) {
			status, _ := doRequest(t, app, tt.method, tt.path, `{}`, "")
			if status != fiber.StatusUnauthorized {
				t.Fatalf("missing token: status = %d, want 401", status)
			}
		})
		t.Run(tt.method+" "+tt.path+" invalid token", func(t *testing.T) {
			status, _ := doRequest(t, app, tt.method, tt.path, `{}`, "not-a-real-token")
			if status != fiber.StatusUnauthorized {
				t.Fatalf("invalid token: status = %d, want 401", status)
			}
		})
		t.Run(tt.method+" "+tt.path+" valid token", func(t *testing.T) {
			status, _ := doRequest(t, app, tt.method, tt.path, `{"name":"Alice2"}`, token)
			if status == fiber.StatusUnauthorized {
				t.Fatalf("valid token rejected: status = %d", status)
			}
		})
	}
}

func TestList_ServiceError(t *testing.T) {
	svc := &fakeService{err: errors.New("boom")}
	app := newTestApp(t, svc)
	token := issueToken(t)

	status, _ := doRequest(t, app, "GET", "/api/v1/users", "", token)
	if status != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", status)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := &fakeService{err: domain.ErrNotFound}
	app := newTestApp(t, svc)
	token := issueTokenFor(t, "missing")

	status, _ := doRequest(t, app, "GET", "/api/v1/users/missing", "", token)
	if status != fiber.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}

func TestGetByID_Forbidden(t *testing.T) {
	svc := &fakeService{output: sampleOutput()}
	app := newTestApp(t, svc)
	token := issueToken(t) // subject u1

	status, _ := doRequest(t, app, "GET", "/api/v1/users/other", "", token)
	if status != fiber.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func TestGetByID_Forbidden_SelfMismatch(t *testing.T) {
	svc := &fakeService{err: domain.ErrInvalidID}
	app := newTestApp(t, svc)
	token := issueToken(t) // subject u1

	// A malformed id that does not match the caller's subject is denied before
	// validation — ownership is checked first.
	status, _ := doRequest(t, app, "GET", "/api/v1/users/%20", "", token)
	if status != fiber.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func TestGetByID_OK(t *testing.T) {
	svc := &fakeService{output: sampleOutput()}
	app := newTestApp(t, svc)
	token := issueToken(t)

	status, body := doRequest(t, app, "GET", "/api/v1/users/u1", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", status, body)
	}
	var got user.UserOutput
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "u1" {
		t.Errorf("body = %+v", got)
	}
}

func TestList(t *testing.T) {
	svc := &fakeService{list: []user.UserOutput{sampleOutput()}}
	app := newTestApp(t, svc)
	token := issueToken(t)

	status, body := doRequest(t, app, "GET", "/api/v1/users", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	var list []user.UserOutput
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
}

func TestSearch(t *testing.T) {
	svc := &fakeService{results: []user.SearchResultOutput{{User: sampleOutput(), Rank: 15}}}
	app := newTestApp(t, svc)
	token := issueToken(t)

	status, body := doRequest(t, app, "GET", "/api/v1/users/search?q=ali", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", status, body)
	}
	var results []user.SearchResultOutput
	if err := json.Unmarshal(body, &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Rank != 15 {
		t.Fatalf("results = %+v, want one result with rank 15", results)
	}
}

func TestSearch_RequiresAuth(t *testing.T) {
	app := newTestApp(t, &fakeService{})
	status, _ := doRequest(t, app, "GET", "/api/v1/users/search?q=ali", "", "")
	if status != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestUpdate(t *testing.T) {
	svc := &fakeService{output: user.UserOutput{ID: "u1", Name: "Alice Cooper", Email: "alice@example.com", CreatedAt: time.Now()}}
	app := newTestApp(t, svc)
	token := issueToken(t)

	status, body := doRequest(t, app, "PUT", "/api/v1/users/u1", `{"name":"Alice Cooper"}`, token)
	if status != fiber.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", status, body)
	}
}

func TestUpdate_BadBody(t *testing.T) {
	app := newTestApp(t, &fakeService{})
	token := issueToken(t)

	status, _ := doRequest(t, app, "PUT", "/api/v1/users/u1", `{"unknown":"x"}`, token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	svc := &fakeService{err: domain.ErrNotFound}
	app := newTestApp(t, svc)
	token := issueTokenFor(t, "missing")

	status, _ := doRequest(t, app, "PUT", "/api/v1/users/missing", `{"name":"X"}`, token)
	if status != fiber.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}

func TestUpdate_Forbidden(t *testing.T) {
	svc := &fakeService{output: sampleOutput()}
	app := newTestApp(t, svc)
	token := issueToken(t) // subject u1

	status, _ := doRequest(t, app, "PUT", "/api/v1/users/other", `{"name":"X"}`, token)
	if status != fiber.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func TestUpdate_Conflict(t *testing.T) {
	svc := &fakeService{err: domain.ErrDuplicateEmail}
	app := newTestApp(t, svc)
	token := issueToken(t)

	status, _ := doRequest(t, app, "PUT", "/api/v1/users/u1", `{"email":"taken@example.com"}`, token)
	if status != fiber.StatusConflict {
		t.Fatalf("status = %d, want 409", status)
	}
}

func TestUpdate_ValidationError(t *testing.T) {
	svc := &fakeService{err: &domain.ValidationError{Field: "name", Message: "is required"}}
	app := newTestApp(t, svc)
	token := issueToken(t)

	status, body := doRequest(t, app, "PUT", "/api/v1/users/u1", `{"name":""}`, token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", status, body)
	}
	var errBody struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(body, &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody.Error != "validation error: name is required" || errBody.Fields["name"] != "is required" {
		t.Errorf("body = %s, want validation envelope", body)
	}
}

func TestDelete_NoContent(t *testing.T) {
	app := newTestApp(t, &fakeService{})
	token := issueToken(t)

	status, _ := doRequest(t, app, "DELETE", "/api/v1/users/u1", "", token)
	if status != fiber.StatusNoContent {
		t.Fatalf("status = %d, want 204", status)
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := &fakeService{err: domain.ErrNotFound}
	app := newTestApp(t, svc)
	token := issueTokenFor(t, "missing")

	status, _ := doRequest(t, app, "DELETE", "/api/v1/users/missing", "", token)
	if status != fiber.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}

func TestDelete_Forbidden(t *testing.T) {
	app := newTestApp(t, &fakeService{})
	token := issueToken(t) // subject u1

	status, _ := doRequest(t, app, "DELETE", "/api/v1/users/other", "", token)
	if status != fiber.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func TestLogin_RateLimited(t *testing.T) {
	app := newTestApp(t, &fakeService{})

	// Exceed the login attempt limit for the same client IP.
	var last int
	for i := 0; i <= middleware.LoginMaxAttempts; i++ {
		last, _ = doRequest(t, app, "POST", "/api/v1/auth/login",
			`{"email":"a@b.com","password":"wrong"}`, "")
	}
	if last != fiber.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", last)
	}
}

func TestInternalError(t *testing.T) {
	svc := &fakeService{err: errors.New("boom")}
	app := newTestApp(t, svc)

	status, _ := doRequest(t, app, "POST", "/api/v1/users",
		`{"name":"A","email":"a@b.com","password":"secret123"}`, "")
	if status != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", status)
	}
}
