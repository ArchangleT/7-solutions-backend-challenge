// Package utility provides low-level HTTP plumbing shared by the feature
// handlers: request body decoding and domain-error mapping. It depends only on
// Fiber and the domain, so it can be imported by the assembling HTTP server
// without pulling in feature packages.
package utility

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	domain "github.com/7-solutions/backend-challenge/internal/domain/user"
)

// MaxBodyBytes caps request bodies at 1 MiB.
const MaxBodyBytes = 1048576 // 1 << 20 // 1 MiB

// errorBody is the standard error envelope returned by the API.
type errorBody struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

// DecodeBody decodes a single JSON object and rejects unknown fields.
// It returns true on success; on failure it writes the error response and
// returns false.
func DecodeBody(c *fiber.Ctx, dst any) bool {
	if len(c.Body()) > MaxBodyBytes {
		_ = c.Status(fiber.StatusRequestEntityTooLarge).JSON(errorBody{Error: "request body too large"})
		return false
	}

	dec := json.NewDecoder(bytes.NewReader(c.Body()))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(errorBody{Error: "invalid request body"})
		return false
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		_ = c.Status(fiber.StatusBadRequest).JSON(errorBody{Error: "request body must contain a single JSON object"})
		return false
	}
	return true
}

// WriteServiceError translates domain errors into HTTP status codes so the
// application layer stays HTTP-agnostic. Unhandled errors are logged via the
// provided logger.
func WriteServiceError(c *fiber.Ctx, logger *slog.Logger, err error) error {
	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(errorBody{
			Error:  verr.Message,
			Fields: map[string]string{verr.Field: verr.Message},
		})
	}
	switch {
	case errors.Is(err, domain.ErrInvalidID):
		return c.Status(fiber.StatusBadRequest).JSON(errorBody{Error: "invalid user id"})
	case errors.Is(err, domain.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(errorBody{Error: "user not found"})
	case errors.Is(err, domain.ErrDuplicateEmail):
		return c.Status(fiber.StatusConflict).JSON(errorBody{Error: "email already exists"})
	case errors.Is(err, domain.ErrInvalidCredentials):
		return c.Status(fiber.StatusUnauthorized).JSON(errorBody{Error: "invalid credentials"})
	default:
		logger.Error("unhandled service error", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(errorBody{Error: "internal server error"})
	}
}
