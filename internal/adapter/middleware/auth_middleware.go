// Package middleware holds middleware reused across inbound adapters.
// It depends only on application ports and the web framework, never on a
// concrete driver or database.
package middleware

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/7-solutions/backend-challenge/internal/application/ports"
)

// UserIDKey is the request-locals key under which the authenticated user ID is
// stored by Authenticate.
const UserIDKey = "user_id"

// ErrUnauthorized is returned by Authenticate after the 401 response has been
// written, so callers can stop the handler chain without re-writing it.
var ErrUnauthorized = errors.New("unauthorized")

// Authenticate validates the Bearer token in c and stores the authenticated
// user ID in the request locals. On a missing or invalid token it writes the
// 401 response and returns ErrUnauthorized.
func Authenticate(c *fiber.Ctx, tokens ports.TokenManager) error {
	token := bearerToken(c.Get(fiber.HeaderAuthorization))
	if token == "" {
		return unauthorized(c)
	}
	userID, err := tokens.Validate(token)
	if err != nil {
		return unauthorized(c)
	}
	c.Locals(UserIDKey, userID)
	return nil
}

// AuthMiddleware wraps Authenticate for use as a Fiber middleware: on a
// missing or invalid token the 401 response is already written, so it stops
// the chain by returning nil; otherwise it delegates to the next handler.
func AuthMiddleware(tokens ports.TokenManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := Authenticate(c, tokens); err != nil {
			return nil
		}
		return c.Next()
	}
}

func unauthorized(c *fiber.Ctx) error {
	_ = c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	return ErrUnauthorized
}

func bearerToken(header string) string {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
