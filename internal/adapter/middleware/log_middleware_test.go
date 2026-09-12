package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		handler    fiber.Handler
		wantStatus int
	}{
		{"success", "/ok", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) }, fiber.StatusOK},
		{"unmatched route", "/missing", nil, fiber.StatusNotFound},
		{"internal error", "/boom", func(c *fiber.Ctx) error {
			return fiber.NewError(fiber.StatusInternalServerError, "boom")
		}, fiber.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

			app := fiber.New()
			app.Use(LoggingMiddleware(logger))
			if tt.handler != nil {
				app.Get(tt.path, tt.handler)
			}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			got := buf.String()
			for _, want := range []string{
				`msg="http request"`,
				"method=GET",
				"path=" + tt.path,
				"status=" + strconv.Itoa(tt.wantStatus),
				"duration=",
			} {
				if !strings.Contains(got, want) {
					t.Errorf("log output missing %q: %s", want, got)
				}
			}
		})
	}
}
