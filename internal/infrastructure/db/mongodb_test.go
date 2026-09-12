package db

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestConnect_InvalidURI(t *testing.T) {
	if _, err := Connect(context.Background(), "://not-a-uri"); err == nil {
		t.Fatal("Connect() with invalid URI: want error, got nil")
	}
}

func TestConnect_PingFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Connect(ctx, "mongodb://localhost:27017"); err == nil {
		t.Fatal("Connect() with cancelled context: want error, got nil")
	}
}

// TestConnect_Success covers the happy path by stubbing the ping probe, so it
// runs offline. The real integration path is exercised by TestConnect_Live.
func TestConnect_Success(t *testing.T) {
	old := ping
	ping = func(context.Context, *mongo.Client) error { return nil }
	t.Cleanup(func() { ping = old })

	client, err := Connect(context.Background(), "mongodb://localhost:27017")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if client == nil {
		t.Fatal("Connect() returned nil client")
	}
	defer client.Disconnect(context.Background())
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
