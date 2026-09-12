package http

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/7-solutions/backend-challenge/internal/adapter/inbound/http/health-check"
	userh "github.com/7-solutions/backend-challenge/internal/adapter/inbound/http/user"
	"github.com/7-solutions/backend-challenge/internal/adapter/inbound/http/helpers"
	"github.com/7-solutions/backend-challenge/internal/adapter/middleware"
	"github.com/7-solutions/backend-challenge/internal/application/ports"
	"github.com/7-solutions/backend-challenge/internal/application/services/user"
)

// Server holds the HTTP plumbing shared by the feature handlers. It implements
// user.Responder so the feature packages can decode bodies and map errors
// without importing this package.
type Server struct {
	logger *slog.Logger
}

// DecodeBody and WriteServiceError delegate to the shared helpers package so
// the actual plumbing lives outside the assembling http package.
func (s *Server) DecodeBody(c *fiber.Ctx, dst any) bool {
	return helpers.DecodeBody(c, dst)
}

func (s *Server) WriteServiceError(c *fiber.Ctx, err error) error {
	return helpers.WriteServiceError(c, s.logger, err)
}

// apiHandler adapts the feature handlers to the generated ServerInterface
// (see server_openapi.gen.go) so every route is registered through it. The
// protected operations guard each request with middleware.Authenticate.
type apiHandler struct {
	users  *userh.Handler
	tokens ports.TokenManager
}

var _ ServerInterface = (*apiHandler)(nil)

func (h *apiHandler) HealthCheck(c *fiber.Ctx) error {
	return healthcheck.Health(c)
}

func (h *apiHandler) Login(c *fiber.Ctx) error {
	return h.users.Login(c)
}

func (h *apiHandler) RegisterUser(c *fiber.Ctx) error {
	return h.users.Register(c)
}

func (h *apiHandler) ListUsers(c *fiber.Ctx) error {
	if err := middleware.Authenticate(c, h.tokens); err != nil {
		return nil
	}
	return h.users.List(c)
}

func (h *apiHandler) GetUser(c *fiber.Ctx, _ UserID) error {
	if err := middleware.Authenticate(c, h.tokens); err != nil {
		return nil
	}
	return h.users.GetByID(c)
}

func (h *apiHandler) UpdateUser(c *fiber.Ctx, _ UserID) error {
	if err := middleware.Authenticate(c, h.tokens); err != nil {
		return nil
	}
	return h.users.Update(c)
}

func (h *apiHandler) DeleteUser(c *fiber.Ctx, _ UserID) error {
	if err := middleware.Authenticate(c, h.tokens); err != nil {
		return nil
	}
	return h.users.Delete(c)
}

// NewServer builds the Fiber application, registering all routes through the
// oapi-codegen generated ServerInterface (see server_openapi.gen.go).
func NewServer(users user.Service, tokens ports.TokenManager, logger *slog.Logger) *fiber.App {
	s := &Server{logger: logger}

	userH := userh.NewHandler(users, s)

	app := fiber.New(fiber.Config{BodyLimit: helpers.MaxBodyBytes})
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	app.Use(middleware.LoggingMiddleware(logger))

	RegisterHandlers(app, &apiHandler{users: userH, tokens: tokens})

	return app
}
