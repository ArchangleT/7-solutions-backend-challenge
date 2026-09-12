// Package statusdb exposes the collection-level statistics use case used by the
// background tasks, so the user CRUD service stays focused on user flows.
package statusdb

import (
	"context"

	domainstatusdb "github.com/7-solutions/backend-challenge/internal/domain/statusdb"
)

// Service exposes the count use case for the users collection.
type Service struct {
	repo domainstatusdb.Repository
}

// NewService wires the count use case to a repository.
func NewService(repo domainstatusdb.Repository) *Service {
	return &Service{repo: repo}
}

// Count returns the total number of User documents.
func (s *Service) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}
