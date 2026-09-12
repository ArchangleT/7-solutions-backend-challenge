package middleware

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

// LoggingMiddleware logs method, path, status code and execution time.
func LoggingMiddleware(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			// Fiber renders the error response after the middleware chain
			// returns, so the status is carried by the error itself: use the
			// fiber.Error code (or 500) instead of the untouched response.
			var fe *fiber.Error
			if errors.As(err, &fe) {
				status = fe.Code
			} else {
				status = fiber.StatusInternalServerError
			}
		}
		logger.Info("http request",
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"duration", time.Since(start),
		)
		return err
	}
}
