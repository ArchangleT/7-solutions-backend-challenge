// Package middleware holds middleware reused across inbound adapters.
// It depends only on application ports and the web framework, never on a
// concrete driver or database.
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/7-solutions/backend-challenge/internal/application/ports"
)

// UserIDKey is the request-locals key under which the authenticated user ID is
// stored by AuthMiddleware.
const UserIDKey = "user_id"

// AuthMiddleware validates the Bearer token and stores the authenticated user
// ID in the request locals before delegating to the next handler.
func AuthMiddleware(tokens ports.TokenManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := bearerToken(c.Get(fiber.HeaderAuthorization))
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		userID, err := tokens.Validate(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		c.Locals(UserIDKey, userID)
		return c.Next()
	}
}

func bearerToken(header string) string {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
