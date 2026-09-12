// Package user implements the HTTP handlers for the user use cases. The
// handlers depend only on the application service (a driving port) and on a
// small Responder interface provided by the assembling HTTP server, keeping
// this package free of HTTP plumbing such as body decoding and error mapping.
package user

import (
	"github.com/gofiber/fiber/v2"

	appuser "github.com/7-solutions/backend-challenge/internal/application/services/user"
)

// Responder abstracts request decoding and error writing so the handlers can
// stay in this package without importing the assembling http package.
type Responder interface {
	DecodeBody(c *fiber.Ctx, dst any) bool
	WriteServiceError(c *fiber.Ctx, err error) error
}

// Handler wires the user service to Fiber handlers.
type Handler struct {
	Users appuser.Service
	Responder
}

// NewHandler builds the user handlers around a concrete user service.
func NewHandler(users appuser.Service, responder Responder) *Handler {
	return &Handler{Users: users, Responder: responder}
}

// Register handles POST /api/v1/users.
func (h *Handler) Register(c *fiber.Ctx) error {
	var in appuser.CreateUserInput
	if !h.DecodeBody(c, &in) {
		return nil
	}
	out, err := h.Users.Register(c.UserContext(), in)
	if err != nil {
		return h.WriteServiceError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(c *fiber.Ctx) error {
	var in appuser.LoginInput
	if !h.DecodeBody(c, &in) {
		return nil
	}
	out, err := h.Users.Login(c.UserContext(), in)
	if err != nil {
		return h.WriteServiceError(c, err)
	}
	return c.JSON(out)
}

// List handles GET /api/v1/users.
func (h *Handler) List(c *fiber.Ctx) error {
	users, err := h.Users.List(c.UserContext())
	if err != nil {
		return h.WriteServiceError(c, err)
	}
	return c.JSON(users)
}

// GetByID handles GET /api/v1/users/:id.
func (h *Handler) GetByID(c *fiber.Ctx) error {
	out, err := h.Users.GetByID(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.WriteServiceError(c, err)
	}
	return c.JSON(out)
}

// Update handles PUT /api/v1/users/:id.
func (h *Handler) Update(c *fiber.Ctx) error {
	var in appuser.UpdateUserInput
	if !h.DecodeBody(c, &in) {
		return nil
	}
	out, err := h.Users.Update(c.UserContext(), c.Params("id"), in)
	if err != nil {
		return h.WriteServiceError(c, err)
	}
	return c.JSON(out)
}

// Delete handles DELETE /api/v1/users/:id.
func (h *Handler) Delete(c *fiber.Ctx) error {
	if err := h.Users.Delete(c.UserContext(), c.Params("id")); err != nil {
		return h.WriteServiceError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
