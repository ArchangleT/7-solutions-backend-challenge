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

// SearchUsersInput carries the raw free-text query submitted for search.
type SearchUsersInput struct {
	Query string `json:"q"`
}

// UserOutput is the public representation of a user, safe to expose over the API.
type UserOutput struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// SearchResultOutput pairs a matched user with its relevance score. Higher
// rank means a better match; results are returned best-match first.
type SearchResultOutput struct {
	User UserOutput `json:"user"`
	Rank int64      `json:"rank"`
}

// LoginOutput bundles the issued token with the authenticated user.
type LoginOutput struct {
	Token string     `json:"token"`
	User  UserOutput `json:"user"`
}
