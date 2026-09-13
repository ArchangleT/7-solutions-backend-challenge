package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/ArchangleT/7-solutions-backend-challenge/internal/adapter/outbound/jwt"
)

// TestAuthenticate drives the token extraction and validation logic of the gRPC
// auth helper directly, covering the metadata edge cases the transport-level
// bufconn tests cannot reach (missing metadata map, empty bearer token, and a
// non-Bearer authorization header).
func TestAuthenticate(t *testing.T) {
	tm := jwt.NewTokenManager("test-secret")
	valid, err := tm.Generate("u1")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		ctx         context.Context
		wantSubject string
		wantCode    codes.Code
	}{
		{
			name:     "no metadata on context",
			ctx:      context.Background(),
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "authorization key missing",
			ctx:      metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-other", "1")),
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "empty bearer token",
			ctx:      metadata.NewIncomingContext(context.Background(), metadata.Pairs(authorizationKey, "Bearer ")),
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "non bearer authorization header",
			ctx:      metadata.NewIncomingContext(context.Background(), metadata.Pairs(authorizationKey, "Basic abc")),
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "invalid token",
			ctx:      metadata.NewIncomingContext(context.Background(), metadata.Pairs(authorizationKey, "Bearer not-a-jwt")),
			wantCode: codes.Unauthenticated,
		},
		{
			name:        "valid token",
			ctx:         metadata.NewIncomingContext(context.Background(), metadata.Pairs(authorizationKey, "Bearer "+valid)),
			wantSubject: "u1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject, err := authenticate(tt.ctx, tm)
			if tt.wantCode != codes.OK {
				if got := status.Code(err); got != tt.wantCode {
					t.Fatalf("authenticate() code = %v, want %v (err=%v)", got, tt.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("authenticate() error = %v, want nil", err)
			}
			if subject != tt.wantSubject {
				t.Errorf("authenticate() = %q, want %q", subject, tt.wantSubject)
			}
		})
	}
}
