package healthcheck

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestHealth(t *testing.T) {
	app := fiber.New()
	app.Get("/health", Health)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"plain", "/health", fiber.StatusOK, `{"status":"ok"}`},
		{"with query", "/health?x=1", fiber.StatusOK, `{"status":"ok"}`},
		{"unknown path", "/healthz", fiber.StatusNotFound, ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, tt.path, nil))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			raw, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantBody != "" && string(raw) != tt.wantBody {
				t.Errorf("body = %q, want %q", raw, tt.wantBody)
			}
		})
	}
}
