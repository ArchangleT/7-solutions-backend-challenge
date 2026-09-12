// Package bcrypt implements the PasswordHasher port using bcrypt.
package bcrypt

import bcryptgo "golang.org/x/crypto/bcrypt"

// Hasher satisfies ports.PasswordHasher.
type Hasher struct{}

func New() *Hasher { return &Hasher{} }

// Hash returns a bcrypt hash of the given password.
func (h *Hasher) Hash(password string) (string, error) {
	b, err := bcryptgo.GenerateFromPassword([]byte(password), bcryptgo.DefaultCost)
	return string(b), err
}

// Compare reports whether the plain password matches the bcrypt hash.
func (h *Hasher) Compare(hashed, plain string) bool {
	return bcryptgo.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
