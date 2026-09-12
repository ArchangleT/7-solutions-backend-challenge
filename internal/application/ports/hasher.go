package ports

// PasswordHasher is the port for hashing and comparing plain-text passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hashed, plain string) bool
}
