package mongodb

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	domain "github.com/7-solutions/backend-challenge/internal/domain/user"
	"github.com/7-solutions/backend-challenge/internal/infrastructure/db"
)

// TestUserRepositoryIntegration exercises the MongoDB adapter against a real
// database. It is skipped unless TEST_MONGO_URI is set, e.g.:
//
//	docker compose up -d mongo
//	TEST_MONGO_URI=mongodb://localhost:27017 go test ./internal/adapter/outbound/mongodb/ -run Integration -v
func TestUserRepositoryIntegration(t *testing.T) {
	uri := os.Getenv("TEST_MONGO_URI")
	if uri == "" {
		t.Skip("TEST_MONGO_URI not set; skipping MongoDB integration test")
	}

	client, err := db.Connect(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	// use an isolated collection per run to keep the test idempotent
	dbName := "userdb_integration"
	coll := client.Database(dbName).Collection(usersCollection)
	_ = coll.Drop(context.Background())
	t.Cleanup(func() { _ = coll.Drop(context.Background()) })

	repo := NewUserRepository(coll)
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	t.Run("create + find by id + count", func(t *testing.T) {
		u := &domain.User{Name: "Alice", Email: "alice@example.com", Password: "hash", CreatedAt: time.Now().UTC()}
		if err := repo.Create(ctx, u); err != nil {
			t.Fatal(err)
		}
		if u.ID == "" {
			t.Fatal("Create() did not assign an ID")
		}

		got, err := repo.FindByID(ctx, u.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Email != u.Email {
			t.Errorf("FindByID() email = %q, want %q", got.Email, u.Email)
		}

		n, err := repo.Count(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if n < 1 {
			t.Errorf("Count() = %d, want >= 1", n)
		}
	})

	t.Run("duplicate email is rejected", func(t *testing.T) {
		u := &domain.User{Name: "Bob", Email: "alice@example.com", Password: "hash", CreatedAt: time.Now().UTC()}
		err := repo.Create(ctx, u)
		if !errors.Is(err, domain.ErrDuplicateEmail) {
			t.Fatalf("Create() error = %v, want ErrDuplicateEmail", err)
		}
	})

	t.Run("update name and email", func(t *testing.T) {
		u, err := repo.FindByEmail(ctx, "alice@example.com")
		if err != nil {
			t.Fatal(err)
		}
		u.Name = "Alice Cooper"
		u.Email = "alice.cooper@example.com"
		if err := repo.Update(ctx, u); err != nil {
			t.Fatal(err)
		}

		got, err := repo.FindByEmail(ctx, "alice.cooper@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != "Alice Cooper" {
			t.Errorf("name = %q, want Alice Cooper", got.Name)
		}
	})

	t.Run("list returns the users", func(t *testing.T) {
		users, err := repo.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 {
			t.Errorf("len(List()) = %d, want 1", len(users))
		}
	})

	t.Run("update missing user is not found", func(t *testing.T) {
		err := repo.Update(ctx, &domain.User{
			ID:        "0123456789abcdef01234567",
			Name:      "Ghost",
			Email:     "ghost@example.com",
			CreatedAt: time.Now().UTC(),
		})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("Update() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("delete user", func(t *testing.T) {
		u, err := repo.FindByEmail(ctx, "alice.cooper@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.Delete(ctx, u.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.FindByID(ctx, u.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("FindByID() after delete error = %v, want ErrNotFound", err)
		}
		if err := repo.Delete(ctx, u.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("Delete() again error = %v, want ErrNotFound", err)
		}
	})
}
