// Command api is the composition root of the application. It wires the
// adapters to the application ports: MongoDB client, JWT token manager, bcrypt
// hasher and the Fiber HTTP server. This is where infrastructure is assembled
// into the application.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/7-solutions/backend-challenge/config"
	httpapi "github.com/7-solutions/backend-challenge/internal/adapter/inbound/http"
	"github.com/7-solutions/backend-challenge/internal/adapter/outbound/bcrypt"
	"github.com/7-solutions/backend-challenge/internal/adapter/outbound/jwt"
	outmongodb "github.com/7-solutions/backend-challenge/internal/adapter/outbound/mongodb"
	"github.com/7-solutions/backend-challenge/internal/application/services/statusdb"
	"github.com/7-solutions/backend-challenge/internal/application/services/user"
	"github.com/7-solutions/backend-challenge/internal/infrastructure/db"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := db.Connect(ctx, cfg.MongoURI)
	if err != nil {
		logger.Error("failed to connect to mongodb", "uri", cfg.MongoURI, "error", err)
		os.Exit(1)
	}
	defer client.Disconnect(context.WithoutCancel(ctx))

	userRepo := outmongodb.NewUserRepository(client.Database(cfg.MongoDatabase).Collection("users"))
	if err := userRepo.EnsureIndexes(ctx); err != nil {
		logger.Error("failed to ensure indexes", "error", err)
		os.Exit(1)
	}

	tokenManager := jwt.NewTokenManager(cfg.JWTSecret)
	userService := user.NewService(userRepo, bcrypt.New(), tokenManager)

	// Concurrency task: a background goroutine that logs the total number of
	// users in the database every 10 seconds.
	go countUsersLoop(ctx, statusdb.NewService(userRepo), logger)

	app := httpapi.NewServer(userService, tokenManager, logger)
	addr := ":" + cfg.Port

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", addr)
		serveErr <- app.Listen(addr)
	}()

	select {
	case err := <-serveErr:
		logger.Error("http server exited", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	if err := app.Shutdown(); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
	logger.Info("server stopped")
}

// countUsersLoop polls the user count every 10 seconds until the context is
// cancelled. It is the implementation of the requirement's concurrency task.
func countUsersLoop(ctx context.Context, svc *statusdb.Service, logger *slog.Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	logger.Info("user count task started")
	for {
		select {
		case <-ctx.Done():
			logger.Info("user count task stopped")
			return
		case <-ticker.C:
			total, err := svc.Count(ctx)
			if err != nil {
				logger.Error("failed to count users", "error", err)
				continue
			}
			logger.Info("total users", "count", total)
		}
	}
}
