package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	domain "github.com/7-solutions/backend-challenge/internal/domain/user"
)

// ---- fakes implementing the application ports ----

type memoryRepo struct {
	users      map[string]*domain.User
	idsByEmail map[string]string
	err        error
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{users: map[string]*domain.User{}, idsByEmail: map[string]string{}}
}

func (r *memoryRepo) Create(_ context.Context, u *domain.User) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.idsByEmail[u.Email]; ok {
		return domain.ErrDuplicateEmail
	}
	if u.ID == "" {
		u.ID = fmt.Sprintf("u%03d", len(r.users)+1)
	}
	r.users[u.ID] = u
	r.idsByEmail[u.Email] = u.ID
	return nil
}

func (r *memoryRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (r *memoryRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	id, ok := r.idsByEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return r.users[id], nil
}

func (r *memoryRepo) List(_ context.Context) ([]domain.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	users := make([]domain.User, 0, len(r.users))
	for _, u := range r.users {
		users = append(users, *u)
	}
	return users, nil
}

func (r *memoryRepo) Update(_ context.Context, u *domain.User) error {
	if r.err != nil {
		return r.err
	}
	old, ok := r.users[u.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if id, exists := r.idsByEmail[u.Email]; exists && id != u.ID {
		return domain.ErrDuplicateEmail
	}
	delete(r.idsByEmail, old.Email)
	r.users[u.ID] = u
	r.idsByEmail[u.Email] = u.ID
	return nil
}

func (r *memoryRepo) Delete(_ context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	u, ok := r.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(r.users, id)
	delete(r.idsByEmail, u.Email)
	return nil
}

type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) { return "hash(" + password + ")", nil }
func (fakeHasher) Compare(hashed, plain string) bool    { return hashed == "hash("+plain+")" }

type fakeTokens struct{}

func (fakeTokens) Generate(userID string) (string, error) { return "token-" + userID, nil }
func (fakeTokens) Validate(token string) (string, error) {
	if strings.HasPrefix(token, "token-") {
		return strings.TrimPrefix(token, "token-"), nil
	}
	return "", errors.New("invalid token")
}

// ---- tests ----

func newTestService(t *testing.T) (*service, *memoryRepo) {
	t.Helper()
	repo := newMemoryRepo()
	return NewService(repo, fakeHasher{}, fakeTokens{}).(*service), repo
}

func seedUser(repo *memoryRepo, name, email string) *domain.User {
	u := &domain.User{Name: name, Email: email, Password: "hash(secret123)", CreatedAt: time.Now()}
	_ = repo.Create(context.Background(), u)
	return u
}

func TestRegister_Success(t *testing.T) {
	svc, _ := newTestService(t)

	out, err := svc.Register(context.Background(), CreateUserInput{
		Name: "  Alice  ", Email: "  ALICE@Example.COM ", Password: "secret123",
	})

	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if out.Name != "Alice" {
		t.Errorf("Name = %q, want Alice", out.Name)
	}
	if out.Email != "alice@example.com" {
		t.Errorf("Email = %q, want lowercased alice@example.com", out.Email)
	}
	if out.ID == "" {
		t.Error("ID not assigned")
	}
	if out.CreatedAt.IsZero() {
		t.Error("CreatedAt not set")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, repo := newTestService(t)
	seedUser(repo, "Bob", "bob@example.com")

	_, err := svc.Register(context.Background(), CreateUserInput{
		Name: "Bob Jr", Email: "BOB@example.com", Password: "secret123",
	})
	if !errors.Is(err, domain.ErrDuplicateEmail) {
		t.Fatalf("Register() error = %v, want ErrDuplicateEmail", err)
	}
}

func TestRegister_Validation(t *testing.T) {
	svc, _ := newTestService(t)

	tests := []struct {
		name  string
		input CreateUserInput
		field string
	}{
		{"empty name", CreateUserInput{Name: " ", Email: "a@b.com", Password: "secret123"}, "name"},
		{"invalid email", CreateUserInput{Name: "A", Email: "not-an-email", Password: "secret123"}, "email"},
		{"short password", CreateUserInput{Name: "A", Email: "a@b.com", Password: "short"}, "password"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), tt.input)
			var verr *domain.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("Register() error = %v, want *ValidationError", err)
			}
			if verr.Field != tt.field {
				t.Errorf("field = %q, want %q", verr.Field, tt.field)
			}
		})
	}
}

func TestRegister_HashesPassword(t *testing.T) {
	svc, repo := newTestService(t)

	_, err := svc.Register(context.Background(), CreateUserInput{Name: "A", Email: "a@b.com", Password: "secret123"})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := repo.FindByEmail(context.Background(), "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Password == "secret123" {
		t.Error("password stored in plain text")
	}
	if !strings.HasPrefix(stored.Password, "hash(") {
		t.Errorf("password = %q, want hashed", stored.Password)
	}
}

func TestLogin_Success(t *testing.T) {
	svc, repo := newTestService(t)
	seedUser(repo, "Alice", "alice@example.com")

	out, err := svc.Login(context.Background(), LoginInput{Email: "alice@example.com", Password: "secret123"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if out.Token != "token-"+out.User.ID {
		t.Errorf("Token = %q, want token-<id>", out.Token)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc, repo := newTestService(t)
	seedUser(repo, "Alice", "alice@example.com")

	// unknown email
	_, err := svc.Login(context.Background(), LoginInput{Email: "nobody@example.com", Password: "secret123"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("unknown email: error = %v, want ErrInvalidCredentials", err)
	}

	// wrong password
	_, err = svc.Login(context.Background(), LoginInput{Email: "alice@example.com", Password: "wrong-password"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("wrong password: error = %v, want ErrInvalidCredentials", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	svc, repo := newTestService(t)
	u := seedUser(repo, "Alice", "alice@example.com")

	out, err := svc.GetByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != u.ID || out.Email != u.Email {
		t.Errorf("GetByID() = %+v, want id %q", out, u.ID)
	}
}

func TestGetByID_Errors(t *testing.T) {
	svc, repo := newTestService(t)
	seedUser(repo, "Alice", "alice@example.com")

	if _, err := svc.GetByID(context.Background(), ""); !errors.Is(err, domain.ErrInvalidID) {
		t.Errorf("empty id: error = %v, want ErrInvalidID", err)
	}
	if _, err := svc.GetByID(context.Background(), "nonexistent"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("missing id: error = %v, want ErrNotFound", err)
	}
}

func TestList(t *testing.T) {
	svc, repo := newTestService(t)
	seedUser(repo, "Alice", "alice@example.com")
	seedUser(repo, "Bob", "bob@example.com")

	out, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(out))
	}
}

func TestUpdate_Success(t *testing.T) {
	svc, repo := newTestService(t)
	u := seedUser(repo, "Alice", "alice@example.com")

	name := "Alice Cooper"
	email := "alice.cooper@example.com"
	out, err := svc.Update(context.Background(), u.ID, UpdateUserInput{Name: &name, Email: &email})
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != name || out.Email != email {
		t.Errorf("Update() = %+v, want name %q email %q", out, name, email)
	}
}

func TestUpdate_NoFieldsProvided(t *testing.T) {
	svc, repo := newTestService(t)
	u := seedUser(repo, "Alice", "alice@example.com")

	_, err := svc.Update(context.Background(), u.ID, UpdateUserInput{})
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("Update() error = %v, want *ValidationError", err)
	}
}

func TestUpdate_EmailConflict(t *testing.T) {
	svc, repo := newTestService(t)
	seedUser(repo, "Bob", "bob@example.com")
	u := seedUser(repo, "Alice", "alice@example.com")

	email := "bob@example.com"
	_, err := svc.Update(context.Background(), u.ID, UpdateUserInput{Email: &email})
	if !errors.Is(err, domain.ErrDuplicateEmail) {
		t.Fatalf("Update() error = %v, want ErrDuplicateEmail", err)
	}
}

func TestUpdate_SameEmailAllowed(t *testing.T) {
	svc, repo := newTestService(t)
	u := seedUser(repo, "Alice", "alice@example.com")

	email := "alice@example.com"
	if _, err := svc.Update(context.Background(), u.ID, UpdateUserInput{Email: &email}); err != nil {
		t.Fatalf("Update() same email error = %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	svc, repo := newTestService(t)
	u := seedUser(repo, "Alice", "alice@example.com")

	if err := svc.Delete(context.Background(), u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.FindByID(context.Background(), u.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("user still exists after Delete(): %v", err)
	}
}

func TestDelete_Errors(t *testing.T) {
	svc, _ := newTestService(t)

	if err := svc.Delete(context.Background(), ""); !errors.Is(err, domain.ErrInvalidID) {
		t.Errorf("empty id: error = %v, want ErrInvalidID", err)
	}
	if err := svc.Delete(context.Background(), "nonexistent"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("missing id: error = %v, want ErrNotFound", err)
	}
}
