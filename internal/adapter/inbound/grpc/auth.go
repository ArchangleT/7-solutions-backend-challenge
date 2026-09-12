package grpc

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/7-solutions/backend-challenge/internal/application/ports"
	domain "github.com/7-solutions/backend-challenge/internal/domain/user"
)

const (
	// authorizationKey is the incoming metadata key carrying the Bearer token.
	authorizationKey = "authorization"
	// userIDContextKey stores the authenticated user ID on the request context.
	userIDContextKey contextKey = "grpc_user_id"
)

type contextKey string

// tokenAuth returns a unary interceptor that requires a valid Bearer token in
// the incoming metadata for GetUser (the protected operation). CreateUser
// stays public. The authenticated user ID is stored on the context.
func tokenAuth(tokens ports.TokenManager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod == userv1GetUserFullMethod {
			userID, err := authenticate(ctx, tokens)
			if err != nil {
				return nil, err
			}
			ctx = context.WithValue(ctx, userIDContextKey, userID)
		}
		return handler(ctx, req)
	}
}

func authenticate(ctx context.Context, tokens ports.TokenManager) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", unauthorized()
	}
	values := md.Get(authorizationKey)
	if len(values) == 0 {
		return "", unauthorized()
	}

	token := strings.TrimSpace(strings.TrimPrefix(values[len(values)-1], "Bearer "))
	if token == "" {
		return "", unauthorized()
	}

	userID, err := tokens.Validate(token)
	if err != nil || strings.TrimSpace(userID) == "" {
		return "", unauthorized()
	}
	return userID, nil
}

func unauthorized() error {
	return status.Error(codes.Unauthenticated, "missing or invalid authorization token")
}

const userv1GetUserFullMethod = "/userv1.UserService/GetUser"

// toServiceError translates domain errors into gRPC status codes. Unknown
// errors become codes.Internal; their message is kept for debuggability.
func toServiceError(err error) error {
	var validationErr *domain.ValidationError
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrDuplicateEmail):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidID), errors.As(err, &validationErr):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
