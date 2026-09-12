package http

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/7-solutions/backend-challenge/internal/adapter/inbound/http/healthcheck"
	userh "github.com/7-solutions/backend-challenge/internal/adapter/inbound/http/user"
	"github.com/7-solutions/backend-challenge/internal/adapter/inbound/http/utility"
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

// DecodeBody and WriteServiceError delegate to the shared utility package so
// the actual plumbing lives outside the assembling http package.
func (s *Server) DecodeBody(c *fiber.Ctx, dst any) bool {
	return utility.DecodeBody(c, dst)
}

func (s *Server) WriteServiceError(c *fiber.Ctx, err error) error {
	return utility.WriteServiceError(c, s.logger, err)
}

// NewServer builds the Fiber application with routing and middleware applied.
func NewServer(users user.Service, tokens ports.TokenManager, logger *slog.Logger) *fiber.App {
	s := &Server{logger: logger}

	userH := userh.NewHandler(users, s)

	app := fiber.New(fiber.Config{BodyLimit: utility.MaxBodyBytes})
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	app.Use(middleware.LoggingMiddleware(logger))

	app.Get("/health", healthcheck.Health)
	app.Post("/api/v1/users", userH.Register)
	app.Post("/api/v1/auth/login", userH.Login)

	usersGroup := app.Group("/api/v1/users", middleware.AuthMiddleware(tokens))
	usersGroup.Get("/", userH.List)
	usersGroup.Get("/:id", userH.GetByID)
	usersGroup.Put("/:id", userH.Update)
	usersGroup.Delete("/:id", userH.Delete)

	return app
}
