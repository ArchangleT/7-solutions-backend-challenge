// Command api is the composition root of the application. It wires the
// adapters to the application ports: MongoDB client, JWT token manager, bcrypt
// hasher and the HTTP + gRPC inbound servers. This is where infrastructure is
// assembled into the application.
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/7-solutions/backend-challenge/config"
	grpcapi "github.com/7-solutions/backend-challenge/internal/adapter/inbound/grpc"
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
		logger.Error("failed to connect to mongodb", "uri", redactURI(cfg.MongoURI), "error", err)
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
	go countUsersLoop(ctx, statusdb.NewService(userRepo), logger, 10*time.Second)

	app := httpapi.NewServer(userService, tokenManager, logger)
	httpAddr := ":" + cfg.Port

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Error("failed to listen for grpc", "addr", cfg.GRPCPort, "error", err)
		os.Exit(1)
	}
	grpcSrv := grpcapi.NewServer(userService, tokenManager)

	serveErr := make(chan error, 2)
	go func() {
		logger.Info("http server listening", "addr", httpAddr)
		serveErr <- app.Listen(httpAddr)
	}()
	go func() {
		logger.Info("grpc server listening", "addr", lis.Addr().String())
		serveErr <- grpcSrv.Serve(lis)
	}()

	select {
	case err := <-serveErr:
		logger.Error("server exited", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	if err := app.Shutdown(); err != nil {
		logger.Error("http graceful shutdown failed", "error", err)
	}
	grpcSrv.GracefulStop()
	logger.Info("servers stopped")
}

// countUsersLoop polls the user count at the given interval until the context is
// cancelled. It is the implementation of the requirement's concurrency task.
func countUsersLoop(ctx context.Context, svc *statusdb.Service, logger *slog.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
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

// redactURI strips any userinfo credentials out of a connection URI so they
// never end up in logs.
func redactURI(uri string) string {
	scheme := strings.Index(uri, "://")
	if scheme < 0 {
		return uri
	}
	rest := uri[scheme+3:]
	at := strings.LastIndex(rest, "@")
	if at < 0 {
		return uri
	}
	return uri[:scheme+3] + "***@" + rest[at+1:]
}
