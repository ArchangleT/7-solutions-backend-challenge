package ports

// TokenManager is the port for issuing and validating access tokens.
type TokenManager interface {
	Generate(userID string) (string, error)
	Validate(token string) (string, error)
}
