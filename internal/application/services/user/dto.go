package user

import "time"

// CreateUserInput carries the data needed to register a new user.
type CreateUserInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginInput carries the credentials used to authenticate a user.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateUserInput carries the optional fields to update. Pointers discriminate
// between "not provided" and "provided but empty".
type UpdateUserInput struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

// UserOutput is the public representation of a user, safe to expose over the API.
type UserOutput struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginOutput bundles the issued token with the authenticated user.
type LoginOutput struct {
	Token string     `json:"token"`
	User  UserOutput `json:"user"`
}
