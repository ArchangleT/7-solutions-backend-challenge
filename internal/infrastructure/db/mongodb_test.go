package db

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestConnect(t *testing.T) {
	errPing := errors.New("ping failed")

	tests := []struct {
		name       string
		uri        string
		cancelCtx  bool
		ping       func(context.Context, *mongo.Client) error
		wantOK     bool
		wantErrSub string
		wantErrIs  error
	}{
		{
			name:       "invalid uri",
			uri:        "://not-a-uri",
			ping:       func(context.Context, *mongo.Client) error { return nil },
			wantOK:     false,
			wantErrSub: "connect mongo client",
		},
		{
			name:       "ping failure wraps sentinel",
			uri:        "mongodb://localhost:27017",
			ping:       func(context.Context, *mongo.Client) error { return errPing },
			wantOK:     false,
			wantErrSub: "ping mongo server",
			wantErrIs:  errPing,
		},
		{
			name:      "cancelled context aborts ping",
			uri:       "mongodb://localhost:27017",
			cancelCtx: true,
			ping: func(ctx context.Context, _ *mongo.Client) error {
				<-ctx.Done()
				return ctx.Err()
			},
			wantOK: false,
		},
		{
			name:   "success",
			uri:    "mongodb://localhost:27017",
			ping:   func(context.Context, *mongo.Client) error { return nil },
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := ping
			ping = tt.ping
			t.Cleanup(func() { ping = old })

			ctx := context.Background()
			if tt.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			client, err := Connect(ctx, tt.uri)
			if tt.wantOK {
				if err != nil {
					t.Fatalf("Connect() error = %v, want nil", err)
				}
				if client == nil {
					t.Fatal("Connect() returned nil client")
				}
				if err := client.Disconnect(context.Background()); err != nil {
					t.Fatalf("Disconnect() error = %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("Connect() error = nil, want error")
			}
			if tt.wantErrSub != "" && !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Errorf("Connect() error = %q, want substring %q", err, tt.wantErrSub)
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("Connect() error = %v, want errors.Is(%v)", err, tt.wantErrIs)
			}
		})
	}
}

// TestConnect_Live is an opt-in integration test that requires a live MongoDB
// (see `make test-integration`). It exercises the real wire protocol.
func TestConnect_Live(t *testing.T) {
	uri := os.Getenv("TEST_MONGO_URI")
	if uri == "" {
		t.Skip("TEST_MONGO_URI not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := Connect(ctx, uri)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Disconnect(ctx)
}
