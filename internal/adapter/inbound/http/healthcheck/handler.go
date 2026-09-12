// Package healthcheck exposes the liveness endpoint. It depends only on the
// web framework and never on application use cases or infrastructure.
package healthcheck

import "github.com/gofiber/fiber/v2"

// Health answers the /health probe with a fixed JSON body.
func Health(c *fiber.Ctx) error {
	return c.JSON(map[string]string{"status": "ok"})
}
