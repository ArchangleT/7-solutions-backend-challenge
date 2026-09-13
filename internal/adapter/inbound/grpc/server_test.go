package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	grpcapi "github.com/7-solutions/backend-challenge/internal/adapter/inbound/grpc"
	grpcuserv1 "github.com/7-solutions/backend-challenge/internal/adapter/inbound/grpc/userv1"
	"github.com/7-solutions/backend-challenge/internal/adapter/outbound/jwt"
	"github.com/7-solutions/backend-challenge/internal/application/services/user"
	domain "github.com/7-solutions/backend-challenge/internal/domain/user"
)

const testSecret = "test-secret"

// fakeService implements user.Service for handler-level tests.
type fakeService struct {
	output user.UserOutput
	err    error
}

func (f *fakeService) Register(_ context.Context, _ user.CreateUserInput) (user.UserOutput, error) {
	return f.output, f.err
}

func (f *fakeService) Login(_ context.Context, _ user.LoginInput) (user.LoginOutput, error) {
	return user.LoginOutput{}, f.err
}

func (f *fakeService) GetByID(_ context.Context, _ string) (user.UserOutput, error) {
	return f.output, f.err
}

func (f *fakeService) List(_ context.Context) ([]user.UserOutput, error) {
	return nil, f.err
}

func (f *fakeService) Update(_ context.Context, _ string, _ user.UpdateUserInput) (user.UserOutput, error) {
	return f.output, f.err
}

func (f *fakeService) Delete(_ context.Context, _ string) error { return f.err }

// newTestClient starts an in-process gRPC server wired through the adapter's
// NewServer (auth interceptor included) and returns a connected client plus a
// stop function.
func newTestClient(t *testing.T, svc user.Service) (grpcuserv1.UserServiceClient, func()) {
	t.Helper()

	lis := bufconn.Listen(1 << 20)
	srv := grpcapi.NewServer(svc, jwt.NewTokenManager(testSecret))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return grpcuserv1.NewUserServiceClient(conn), func() {}
}

func TestCreateUser(t *testing.T) {
	want := user.UserOutput{ID: "u1", Name: "Alice", Email: "alice@example.com", CreatedAt: time.Date(2026, 9, 12, 7, 0, 0, 0, time.UTC)}
	client, _ := newTestClient(t, &fakeService{output: want})

	got, err := client.CreateUser(context.Background(), &grpcuserv1.CreateUserRequest{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if got.GetId() != "u1" || got.GetName() != "Alice" || got.GetEmail() != "alice@example.com" {
		t.Errorf("CreateUser() = %+v, want {u1 Alice alice@example.com}", got)
	}
	if got.GetCreatedAt() != "2026-09-12T07:00:00Z" {
		t.Errorf("CreatedAt = %q, want RFC3339", got.GetCreatedAt())
	}
}

func TestCreateUser_ServiceErrorMapped(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"duplicate email", domain.ErrDuplicateEmail, codes.AlreadyExists},
		{"not found", domain.ErrNotFound, codes.NotFound},
		{"invalid id", domain.ErrInvalidID, codes.InvalidArgument},
		{"invalid credentials", domain.ErrInvalidCredentials, codes.Unauthenticated},
		{"validation", &domain.ValidationError{Field: "email", Message: "is invalid"}, codes.InvalidArgument},
		{"internal", errors.New("db is down"), codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := newTestClient(t, &fakeService{err: tc.err})

			_, err := client.CreateUser(context.Background(), &grpcuserv1.CreateUserRequest{})
			if got := status.Code(err); got != tc.code {
				t.Errorf("CreateUser() error code = %v, want %v (err=%v)", got, tc.code, err)
			}
		})
	}
}

func TestGetUser_RequiresToken(t *testing.T) {
	client, _ := newTestClient(t, &fakeService{output: user.UserOutput{ID: "u1", Name: "Alice"}})

	// No metadata -> Unauthenticated.
	_, err := client.GetUser(context.Background(), &grpcuserv1.GetUserRequest{Id: "u1"})
	if got := status.Code(err); got != codes.Unauthenticated {
		t.Errorf("GetUser() without token code = %v, want Unauthenticated", got)
	}

	// Valid token -> succeeds.
	valid, err := jwt.NewTokenManager(testSecret).Generate("u1")
	if err != nil {
		t.Fatal(err)
	}
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+valid))
	got, err := client.GetUser(ctx, &grpcuserv1.GetUserRequest{Id: "u1"})
	if err != nil {
		t.Fatalf("GetUser() with valid token error = %v", err)
	}
	if got.GetId() != "u1" {
		t.Errorf("GetUser() = %+v, want id u1", got)
	}
}

func TestGetUser_ForbiddenOnOtherID(t *testing.T) {
	client, _ := newTestClient(t, &fakeService{output: user.UserOutput{ID: "u1"}})

	valid, err := jwt.NewTokenManager(testSecret).Generate("u1")
	if err != nil {
		t.Fatal(err)
	}
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+valid))
	_, err = client.GetUser(ctx, &grpcuserv1.GetUserRequest{Id: "someone-else"})
	if got := status.Code(err); got != codes.PermissionDenied {
		t.Errorf("GetUser() on other id code = %v, want PermissionDenied (err=%v)", got, err)
	}
}

func TestGetUser_RejectsInvalidToken(t *testing.T) {
	client, _ := newTestClient(t, &fakeService{output: user.UserOutput{ID: "u1"}})

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer not-a-jwt"))
	_, err := client.GetUser(ctx, &grpcuserv1.GetUserRequest{Id: "u1"})
	if got := status.Code(err); got != codes.Unauthenticated {
		t.Errorf("GetUser() with bad token code = %v, want Unauthenticated", got)
	}
}

// spyService records whether the service was reached, letting the tests
// assert the auth interceptor short-circuits before the handler.
type spyService struct {
	fakeService
	called bool
}

func (s *spyService) GetByID(_ context.Context, _ string) (user.UserOutput, error) {
	s.called = true
	return user.UserOutput{}, nil
}

func TestGetUser_UnauthenticatedDoesNotCallService(t *testing.T) {
	svc := &spyService{}
	client, _ := newTestClient(t, svc)

	_, err := client.GetUser(context.Background(), &grpcuserv1.GetUserRequest{Id: "u1"})
	if got := status.Code(err); got != codes.Unauthenticated {
		t.Fatalf("code = %v, want Unauthenticated", got)
	}
	if svc.called {
		t.Error("GetUser reached the service without a token")
	}
}

func TestGetUser_ServiceErrorMapped(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"not found", domain.ErrNotFound, codes.NotFound},
		{"invalid id", domain.ErrInvalidID, codes.InvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := newTestClient(t, &fakeService{err: tc.err})

			valid, err := jwt.NewTokenManager(testSecret).Generate("u1")
			if err != nil {
				t.Fatal(err)
			}
			ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+valid))
			_, err = client.GetUser(ctx, &grpcuserv1.GetUserRequest{Id: "u1"})
			if got := status.Code(err); got != tc.code {
				t.Errorf("GetUser() code = %v, want %v (err=%v)", got, tc.code, err)
			}
		})
	}
}

func TestCreateUser_DoesNotLeakInternalError(t *testing.T) {
	client, _ := newTestClient(t, &fakeService{err: errors.New("mongo connection refused: secret detail")})

	_, err := client.CreateUser(context.Background(), &grpcuserv1.CreateUserRequest{})
	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("CreateUser() code = %v, want Internal", got)
	}
	if msg := status.Convert(err).Message(); msg == "mongo connection refused: secret detail" || msg == "" {
		t.Errorf("CreateUser() leaked internal detail: %q", msg)
	}
}

func TestCreateUser_DoesNotRequireToken(t *testing.T) {
	client, _ := newTestClient(t, &fakeService{
		output: user.UserOutput{ID: "u1", Name: "Alice"},
	})

	// CreateUser is public: no metadata token expected, must not be Unauthenticated.
	if _, err := client.CreateUser(context.Background(), &grpcuserv1.CreateUserRequest{
		Name: "Alice",
	}); err != nil {
		t.Fatalf("CreateUser() without token error = %v", err)
	}
}
