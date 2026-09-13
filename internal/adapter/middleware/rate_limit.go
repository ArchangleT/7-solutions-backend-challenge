package middleware

import (
	"errors"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Login rate limiting: per-client-IP fixed window for the public login
// endpoint, so an attacker cannot brute-force passwords indefinitely.
const (
	// LoginMaxAttempts is the maximum number of login attempts allowed per
	// client IP within the window.
	LoginMaxAttempts = 10
	// LoginWindow is the fixed window length for login rate limiting.
	LoginWindow = time.Minute
)

// ipWindow tracks the attempt count for one client IP inside its window.
type ipWindow struct {
	start time.Time
	count int
}

// RateLimit returns a guard handler that allows at most max requests per
// client IP inside the window. It is meant to be invoked inside a route
// handler (guard pattern): it returns nil without writing a response so the
// caller continues, and returns ErrTooManyRequests after writing a 429
// response (or any other non-nil error the caller wants to stop on). The
// state is held in memory per process, which is appropriate for a
// single-instance MVP.
func RateLimit(max int, window time.Duration) fiber.Handler {
	return rateLimit(max, window, func(c *fiber.Ctx) string { return c.IP() })
}

// rateLimit is RateLimit with an injectable client-IP resolver so tests can
// exercise the per-IP behavior deterministically.
func rateLimit(max int, window time.Duration, clientIP func(*fiber.Ctx) string) fiber.Handler {
	var mu sync.Mutex
	counts := make(map[string]*ipWindow)
	lastSweep := time.Time{}

	// prune removes windows that outlived the current one so the map does not
	// grow unboundedly on a long-running server. It runs at most once per
	// window to keep the sweep cost amortized.
	prune := func(now time.Time) {
		if now.Sub(lastSweep) < window {
			return
		}
		lastSweep = now
		for ip, w := range counts {
			if now.Sub(w.start) >= window {
				delete(counts, ip)
			}
		}
	}

	return func(c *fiber.Ctx) error {
		ip := clientIP(c)
		now := time.Now()

		mu.Lock()
		prune(now)
		w, ok := counts[ip]
		if !ok || now.Sub(w.start) >= window {
			counts[ip] = &ipWindow{start: now, count: 1}
			mu.Unlock()
			return nil
		}
		w.count++
		if w.count > max {
			mu.Unlock()
			_ = c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many requests"})
			return ErrTooManyRequests
		}
		mu.Unlock()
		return nil
	}
}

// ErrTooManyRequests is returned by RateLimit after the 429 response has been
// written, so callers can stop the handler chain without re-writing it.
var ErrTooManyRequests = errors.New("too many requests")
