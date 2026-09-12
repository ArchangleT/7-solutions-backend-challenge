package user

import (
	"errors"
	"fmt"
)

// Sentinel errors shared between layers. Adapters translate framework and
// driver errors to these domain errors; the application layer returns them.
var (
	ErrNotFound           = errors.New("user not found")
	ErrDuplicateEmail     = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidID          = errors.New("invalid user id")
)

// ValidationError describes a single field that failed input validation.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s %s", e.Field, e.Message)
}
