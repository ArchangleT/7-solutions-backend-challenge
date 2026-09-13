// Package middleware holds middleware reused across inbound adapters.
// It depends only on application ports and the web framework, never on a
// concrete driver or database.
package middleware

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ArchangleT/7-solutions-backend-challenge/internal/application/ports"
)

// UserIDKey is the request-locals key under which the authenticated user ID is
// stored by Authenticate.
const UserIDKey = "user_id"

// ErrUnauthorized is returned by Authenticate after the 401 response has been
// written, so callers can stop the handler chain without re-writing it.
var ErrUnauthorized = errors.New("unauthorized")

// ErrForbidden is returned by RequireOwner after the 403 response has been
// written, so callers can stop the handler chain without re-writing it.
var ErrForbidden = errors.New("forbidden")

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

// RequireOwner checks that the client is acting on their own user: the :id
// route parameter must match the authenticated user ID stored by Authenticate.
// On a mismatch it writes the 403 response and returns ErrForbidden.
func RequireOwner(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" || id != CurrentUserID(c) {
		_ = c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		return ErrForbidden
	}
	return nil
}

// CurrentUserID returns the authenticated user ID stored in the request locals
// by Authenticate, or "" when there is none.
func CurrentUserID(c *fiber.Ctx) string {
	id, _ := c.Locals(UserIDKey).(string)
	return id
}

func bearerToken(header string) string {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
