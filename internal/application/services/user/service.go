package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ArchangleT/7-solutions-backend-challenge/internal/application/ports"
	domain "github.com/ArchangleT/7-solutions-backend-challenge/internal/domain/user"
)

const passwordMinLength = 8

// Service is the driving port consumed by the inbound adapters (HTTP handlers).
// It exposes the application use cases without leaking implementation details.
type Service interface {
	Register(ctx context.Context, in CreateUserInput) (UserOutput, error)
	Login(ctx context.Context, in LoginInput) (LoginOutput, error)
	GetByID(ctx context.Context, id string) (UserOutput, error)
	List(ctx context.Context) ([]UserOutput, error)
	Search(ctx context.Context, in SearchUsersInput) ([]SearchResultOutput, error)
	Update(ctx context.Context, id string, in UpdateUserInput) (UserOutput, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo   domain.Repository
	hasher ports.PasswordHasher
	tokens ports.TokenManager
}

// NewService wires the use cases together. The repository, hasher and token
// manager are injected as ports, so the service never depends on a concrete
// driver or framework.
func NewService(repo domain.Repository, hasher ports.PasswordHasher, tokens ports.TokenManager) Service {
	return &service{repo: repo, hasher: hasher, tokens: tokens}
}

func (s *service) Register(ctx context.Context, in CreateUserInput) (UserOutput, error) {
	name := strings.TrimSpace(in.Name)
	email := normalizeEmail(in.Email)

	if name == "" {
		return UserOutput{}, &domain.ValidationError{Field: "name", Message: "is required"}
	}
	if !validEmail(email) {
		return UserOutput{}, &domain.ValidationError{Field: "email", Message: "is invalid"}
	}
	if len(in.Password) < passwordMinLength {
		return UserOutput{}, &domain.ValidationError{Field: "password", Message: "must be at least 8 characters"}
	}

	if _, err := s.repo.FindByEmail(ctx, email); err == nil {
		return UserOutput{}, domain.ErrDuplicateEmail
	} else if !errors.Is(err, domain.ErrNotFound) {
		return UserOutput{}, err
	}

	hashed, err := s.hasher.Hash(in.Password)
	if err != nil {
		return UserOutput{}, err
	}

	u := &domain.User{
		Name:      name,
		Email:     email,
		Password:  hashed,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return UserOutput{}, err
	}
	return toOutput(u), nil
}

func (s *service) Login(ctx context.Context, in LoginInput) (LoginOutput, error) {
	u, err := s.repo.FindByEmail(ctx, normalizeEmail(in.Email))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return LoginOutput{}, domain.ErrInvalidCredentials
		}
		return LoginOutput{}, err
	}

	if !s.hasher.Compare(u.Password, in.Password) {
		return LoginOutput{}, domain.ErrInvalidCredentials
	}

	token, err := s.tokens.Generate(u.ID)
	if err != nil {
		return LoginOutput{}, err
	}
	return LoginOutput{Token: token, User: toOutput(u)}, nil
}

func (s *service) GetByID(ctx context.Context, id string) (UserOutput, error) {
	if strings.TrimSpace(id) == "" {
		return UserOutput{}, domain.ErrInvalidID
	}
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return UserOutput{}, err
	}
	return toOutput(u), nil
}

func (s *service) List(ctx context.Context) ([]UserOutput, error) {
	users, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UserOutput, 0, len(users))
	for i := range users {
		out = append(out, toOutput(&users[i]))
	}
	return out, nil
}

// Search implements the word-prefix, "Top Hits"-ranked search. The repository
// does the matching; this layer parses the raw query into domain search tokens
// and enriches every hit with a relevance score so the API can order by score.
func (s *service) Search(ctx context.Context, in SearchUsersInput) ([]SearchResultOutput, error) {
	query := domain.ParseSearchQuery(in.Query)
	if query.Empty() {
		return []SearchResultOutput{}, nil
	}

	users, err := s.repo.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	out := make([]SearchResultOutput, 0, len(users))
	for i := range users {
		score, _ := query.Score(&users[i])
		out = append(out, SearchResultOutput{User: toOutput(&users[i]), Rank: score})
	}
	return out, nil
}

func (s *service) Update(ctx context.Context, id string, in UpdateUserInput) (UserOutput, error) {
	if strings.TrimSpace(id) == "" {
		return UserOutput{}, domain.ErrInvalidID
	}
	if in.Name == nil && in.Email == nil {
		return UserOutput{}, &domain.ValidationError{Field: "body", Message: "provide at least one of name or email"}
	}

	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return UserOutput{}, err
	}

	if err := s.updateName(u, in.Name); err != nil {
		return UserOutput{}, err
	}
	if err := s.updateEmail(ctx, u, id, in.Email); err != nil {
		return UserOutput{}, err
	}

	if err := s.repo.Update(ctx, u); err != nil {
		return UserOutput{}, err
	}
	return toOutput(u), nil
}

func (s *service) updateName(u *domain.User, name *string) error {
	if name == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*name)
	if trimmed == "" {
		return &domain.ValidationError{Field: "name", Message: "is required"}
	}
	u.Name = trimmed
	return nil
}

func (s *service) updateEmail(ctx context.Context, u *domain.User, id string, email *string) error {
	if email == nil {
		return nil
	}
	normalized := normalizeEmail(*email)
	if !validEmail(normalized) {
		return &domain.ValidationError{Field: "email", Message: "is invalid"}
	}
	if normalized == u.Email {
		return nil
	}
	return s.checkDuplicateEmail(ctx, u, id, normalized)
}

func (s *service) checkDuplicateEmail(ctx context.Context, u *domain.User, id string, email string) error {
	existing, err := s.repo.FindByEmail(ctx, email)
	if err == nil && existing.ID != id {
		return domain.ErrDuplicateEmail
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	u.Email = email
	return nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return domain.ErrInvalidID
	}
	return s.repo.Delete(ctx, id)
}
