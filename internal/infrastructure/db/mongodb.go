// Package db owns the MongoDB client lifecycle: building a client from a URI
// and proving the connection with a ping before handing it back to the
// composition root. Keeping it here instead of inside an adapter makes the
// connection reusable by multiple outbound adapters.
package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const pingTimeout = 5 * time.Second

// ping is a package-level hook so tests can exercise the happy path without a
// live MongoDB server.
var ping = func(ctx context.Context, client *mongo.Client) error {
	return client.Ping(ctx, readpref.Primary())
}

// Connect creates a client from uri and verifies connectivity with a primary
// ping. The returned client is closed by the caller.
func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongo client: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := ping(pingCtx, client); err != nil {
		return nil, fmt.Errorf("ping mongo server: %w", err)
	}
	return client, nil
}
