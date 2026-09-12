package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestHealth(t *testing.T) {
	app := fiber.New()
	app.Get("/health", Health)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/health", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
