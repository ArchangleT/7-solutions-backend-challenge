package mongodb

import (
	"context"
	"testing"

	domain "github.com/7-solutions/backend-challenge/internal/domain/user"
)

// The hex-parse paths below return before any driver call is made, so they can
// be unit-tested with a nil collection. Everything that touches MongoDB is
// exercised by the opt-in integration test in user_repository_integration_test.go.

func TestFindByID_InvalidHexIsNotFound(t *testing.T) {
	repo := &UserRepository{collection: nil}

	if _, err := repo.FindByID(context.Background(), "not-a-hex-id"); err != domain.ErrNotFound {
		t.Fatalf("FindByID() error = %v, want ErrNotFound", err)
	}
}

func TestDelete_InvalidHexIsInvalidID(t *testing.T) {
	repo := &UserRepository{collection: nil}

	if err := repo.Delete(context.Background(), "not-a-hex-id"); err != domain.ErrInvalidID {
		t.Fatalf("Delete() error = %v, want ErrInvalidID", err)
	}
}

func TestUpdate_InvalidHexIsInvalidID(t *testing.T) {
	repo := &UserRepository{collection: nil}

	err := repo.Update(context.Background(), &domain.User{ID: "not-a-hex-id"})
	if err != domain.ErrInvalidID {
		t.Fatalf("Update() error = %v, want ErrInvalidID", err)
	}
}
