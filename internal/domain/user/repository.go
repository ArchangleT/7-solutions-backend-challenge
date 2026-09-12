package user

import "context"

// Repository is the port the application needs to persist User aggregates.
// The domain does not care about the storage technology used to back it
// (MySQL, PostgreSQL, MongoDB or an in-memory mock for tests).
type Repository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context) ([]User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}
