// Package grpc implements the gRPC inbound adapter. It exposes the user
// service over gRPC using the generated UserService stubs (see userv1),
// mirroring the REST endpoints: CreateUser is public, GetUser requires a
// valid Bearer token in the authorization metadata.
package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/7-solutions/backend-challenge/internal/adapter/inbound/grpc/userv1"
	"github.com/7-solutions/backend-challenge/internal/application/ports"
	"github.com/7-solutions/backend-challenge/internal/application/services/user"
)

// Server implements userv1.UserServiceServer, translating between the gRPC
// data types and the application use cases.
type Server struct {
	userv1.UnimplementedUserServiceServer
	users user.Service
}

var _ userv1.UserServiceServer = (*Server)(nil)

// NewServer builds the gRPC server, registering the user service handlers
// behind the given auth interceptor chain.
func NewServer(users user.Service, tokens ports.TokenManager) *grpc.Server {
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(tokenAuth(tokens)))
	userv1.RegisterUserServiceServer(srv, &Server{users: users})
	return srv
}

// CreateUser registers a new user. Public operation: no token required.
func (s *Server) CreateUser(ctx context.Context, in *userv1.CreateUserRequest) (*userv1.User, error) {
	out, err := s.users.Register(ctx, user.CreateUserInput{
		Name:     in.GetName(),
		Email:    in.GetEmail(),
		Password: in.GetPassword(),
	})
	if err != nil {
		return nil, toServiceError(err)
	}
	return toUser(out), nil
}

// GetUser fetches a user by id. Protected operation: the request must carry a
// valid authorization metadata token, and the token subject must match the
// requested id (ownership check).
func (s *Server) GetUser(ctx context.Context, in *userv1.GetUserRequest) (*userv1.User, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok || userID != in.GetId() {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}
	out, err := s.users.GetByID(ctx, in.GetId())
	if err != nil {
		return nil, toServiceError(err)
	}
	return toUser(out), nil
}

func toUser(out user.UserOutput) *userv1.User {
	return &userv1.User{
		Id:        out.ID,
		Name:      out.Name,
		Email:     out.Email,
		CreatedAt: out.CreatedAt.Format(time.RFC3339),
	}
}
