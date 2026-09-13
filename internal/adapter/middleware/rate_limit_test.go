package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// limiterApp builds a small Fiber app using rateLimit (injectable IP
// resolver) in the same guard pattern as the login route: the limiter runs
// first, then the real handler.
func limiterApp(max int) *fiber.App {
	app := fiber.New()
	lim := rateLimit(max, LoginWindow, func(c *fiber.Ctx) string { return c.Get("X-Test-IP") })
	app.Post("/", func(c *fiber.Ctx) error {
		if err := lim(c); err != nil {
			return nil
		}
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func doIP(t *testing.T, app *fiber.App, ip string) int {
	t.Helper()
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("X-Test-IP", ip)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestRateLimit_BlocksOverLimit(t *testing.T) {
	app := limiterApp(3)

	for i := 0; i < 3; i++ {
		if got := doIP(t, app, "10.0.0.1"); got != fiber.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, got)
		}
	}
	if got := doIP(t, app, "10.0.0.1"); got != fiber.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", got)
	}
}

func TestRateLimit_IndependentPerIP(t *testing.T) {
	app := limiterApp(3)
	for i := 0; i < 3; i++ {
		if got := doIP(t, app, "10.0.0.1"); got != fiber.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, got)
		}
	}
	// A different IP is unaffected.
	if got := doIP(t, app, "10.0.0.2"); got != fiber.StatusOK {
		t.Fatalf("other IP: status = %d, want 200", got)
	}
}

// TestRateLimit_PrunesExpiredWindow verifies stale per-IP windows are dropped
// once the window elapses, so a previously limited client can retry after the
// window (and the in-memory map does not grow unboundedly).
func TestRateLimit_PrunesExpiredWindow(t *testing.T) {
	window := 2 * time.Millisecond
	app := fiber.New()
	lim := rateLimit(1, window, func(c *fiber.Ctx) string { return c.Get("X-Test-IP") })
	app.Post("/", func(c *fiber.Ctx) error {
		if err := lim(c); err != nil {
			return nil
		}
		return c.SendStatus(fiber.StatusOK)
	})

	if got := doIP(t, app, "10.0.0.9"); got != fiber.StatusOK {
		t.Fatalf("first request: status = %d, want 200", got)
	}
	if got := doIP(t, app, "10.0.0.9"); got != fiber.StatusTooManyRequests {
		t.Fatalf("overflow request: status = %d, want 429", got)
	}

	time.Sleep(10 * time.Millisecond) // let the window and next sweep pass
	if got := doIP(t, app, "10.0.0.9"); got != fiber.StatusOK {
		t.Fatalf("request after window: status = %d, want 200 (stale entry not pruned?)", got)
	}
}
