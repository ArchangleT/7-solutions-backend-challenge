package user

import (
	"net/mail"
	"strings"

	domain "github.com/7-solutions/backend-challenge/internal/domain/user"
)

func toOutput(u *domain.User) UserOutput {
	return UserOutput{ID: u.ID, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	if email == "" {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return addr.Address == email
}
