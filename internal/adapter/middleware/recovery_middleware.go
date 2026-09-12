package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

// RecoveryMiddleware catches panics inside downstream handlers, logs the
// stack trace and returns 500 Internal Server Error so the process survives.
func RecoveryMiddleware(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					"method", c.Method(),
					"path", c.Path(),
					"error", fmt.Sprintf("%v", r),
					"stack", string(debug.Stack()),
				)
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "internal server error",
				})
			}
		}()
		return c.Next()
	}
}
