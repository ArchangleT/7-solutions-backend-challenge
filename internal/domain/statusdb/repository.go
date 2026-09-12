// Package statusdb holds domain ports for collection-level statistics use
// cases that are orthogonal to the user aggregate.
package statusdb

import "context"

// Repository is the port the application needs for collection-level statistics.
type Repository interface {
	Count(ctx context.Context) (int64, error)
}
