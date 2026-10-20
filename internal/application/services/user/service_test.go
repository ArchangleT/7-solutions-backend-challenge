package user

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

	domain "github.com/ArchangleT/7-solutions-backend-challenge/internal/domain/user"
	portsmocks "github.com/ArchangleT/7-solutions-backend-challenge/internal/mocks/ports"
	"go.uber.org/mock/gomock"
)

var (
	errDB     = errors.New("db down")
	errHash   = errors.New("hash failed")
	errCreate = errors.New("insert failed")
	errJWT    = errors.New("jwt failed")
)

// ---- fake in-memory repository, keeps real errors.Is behaviour ----

type memoryRepo struct {
	users          map[string]*domain.User
	idsByEmail     map[string]string
	err            error
	createErr      error
	findByEmailErr error
	updateErr      error
	searchErr      error
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{users: map[string]*domain.User{}, idsByEmail: map[string]string{}}
}

func (r *memoryRepo) Create(_ context.Context, u *domain.User) error {
	if r.err != nil {
		return r.err
	}
	if r.createErr != nil {
		return r.createErr
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
	if r.findByEmailErr != nil {
		return nil, r.findByEmailErr
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

func (r *memoryRepo) Search(_ context.Context, q domain.SearchQuery) ([]domain.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.searchErr != nil {
		return nil, r.searchErr
	}
	matches := make([]domain.User, 0)
	for _, u := range r.users {
		if _, ok := q.Score(u); ok {
			matches = append(matches, *u)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		si, _ := q.Score(&matches[i])
		sj, _ := q.Score(&matches[j])
		return si > sj
	})
	return matches, nil
}

func (r *memoryRepo) Update(_ context.Context, u *domain.User) error {
	if r.err != nil {
		return r.err
	}
	if r.updateErr != nil {
		return r.updateErr
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

// ---- helpers ----

const hashedSecret = "hash(secret123)"

func newTestService(t *testing.T) (*service, *memoryRepo, *portsmocks.MockPasswordHasher, *portsmocks.MockTokenManager) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := newMemoryRepo()
	hasher := portsmocks.NewMockPasswordHasher(ctrl)
	tokens := portsmocks.NewMockTokenManager(ctrl)
	return NewService(repo, hasher, tokens).(*service), repo, hasher, tokens
}

func seedUser(t *testing.T, repo *memoryRepo, id, name, email string) *domain.User {
	t.Helper()
	u := &domain.User{ID: id, Name: name, Email: email, Password: hashedSecret}
	if err := repo.Create(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	return u
}

func boolp(b bool) *bool { return &b }

func wantValidationErr(t *testing.T, field string, err error) {
	t.Helper()
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %v, want *ValidationError{field: %q}", err, field)
	}
	if verr.Field != field {
		t.Errorf("field = %q, want %q", verr.Field, field)
	}
}

// ---- Register ----

func TestRegister(t *testing.T) {
	tests := []struct {
		name        string
		input       CreateUserInput
		seed        *domain.User
		repoErr     error
		createErr   error
		findByEmail error
		hashErr     error
		wantHash    bool
		wantName    string
		wantEmail   string
		wantField   string
		wantErr     error
	}{
		{
			name:      "success normalizes name and email and hashes password",
			input:     CreateUserInput{Name: "  Alice  ", Email: "  ALICE@Example.COM ", Password: "secret123"},
			wantHash:  true,
			wantName:  "Alice",
			wantEmail: "alice@example.com",
		},
		{
			name:    "duplicate email rejected",
			input:   CreateUserInput{Name: "Bob Jr", Email: "BOB@example.com", Password: "secret123"},
			seed:    &domain.User{ID: "bob", Name: "Bob", Email: "bob@example.com", Password: hashedSecret},
			wantErr: domain.ErrDuplicateEmail,
		},
		{
			name:        "find by email error propagated",
			input:       CreateUserInput{Name: "A", Email: "a@b.com", Password: "secret123"},
			findByEmail: errDB,
			wantErr:     errDB,
		},
		{
			name:     "hasher error propagated",
			input:    CreateUserInput{Name: "A", Email: "a@b.com", Password: "secret123"},
			wantHash: true,
			hashErr:  errHash,
			wantErr:  errHash,
		},
		{
			name:      "repository create error propagated",
			input:     CreateUserInput{Name: "A", Email: "a@b.com", Password: "secret123"},
			wantHash:  true,
			createErr: errCreate,
			wantErr:   errCreate,
		},
		{
			name:      "empty name rejected",
			input:     CreateUserInput{Name: "   ", Email: "a@b.com", Password: "secret123"},
			wantField: "name",
		},
		{
			name:      "invalid email rejected",
			input:     CreateUserInput{Name: "A", Email: "not-an-email", Password: "secret123"},
			wantField: "email",
		},
		{
			name:      "short password rejected",
			input:     CreateUserInput{Name: "A", Email: "a@b.com", Password: "short"},
			wantField: "password",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, hasher, _ := newTestService(t)
			if tt.seed != nil {
				if err := repo.Create(context.Background(), tt.seed); err != nil {
					t.Fatal(err)
				}
			}
			repo.err = tt.repoErr
			repo.createErr = tt.createErr
			repo.findByEmailErr = tt.findByEmail

			if tt.wantHash {
				if tt.hashErr != nil {
					hasher.EXPECT().Hash(tt.input.Password).Return("", tt.hashErr)
				} else {
					hasher.EXPECT().Hash(tt.input.Password).Return(hashedSecret, nil)
				}
			}

			out, err := svc.Register(context.Background(), tt.input)

			if tt.wantField != "" {
				wantValidationErr(t, tt.wantField, err)
				return
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Register() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Register() error = %v", err)
			}
			if out.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", out.Name, tt.wantName)
			}
			if out.Email != tt.wantEmail {
				t.Errorf("Email = %q, want %q", out.Email, tt.wantEmail)
			}
			if out.ID == "" {
				t.Error("ID not assigned")
			}
			if out.CreatedAt.IsZero() {
				t.Error("CreatedAt not set")
			}
			stored, err := repo.FindByEmail(context.Background(), tt.wantEmail)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Password == tt.input.Password {
				t.Error("password stored in plain text")
			}
			if stored.Password != hashedSecret {
				t.Errorf("stored Password = %q, want %q", stored.Password, hashedSecret)
			}
		})
	}
}

// ---- Login ----

func TestLogin(t *testing.T) {
	alice := &domain.User{ID: "alice", Name: "Alice", Email: "alice@example.com", Password: hashedSecret}

	tests := []struct {
		name      string
		input     LoginInput
		seed      *domain.User
		repoErr   error
		compare   *bool
		genErr    error
		wantToken string
		wantErr   error
	}{
		{
			name:      "success issues token",
			input:     LoginInput{Email: "  ALICE@Example.COM ", Password: "secret123"},
			seed:      alice,
			compare:   boolp(true),
			wantToken: "token-alice",
		},
		{
			name:    "unknown email returns invalid credentials",
			input:   LoginInput{Email: "nobody@example.com", Password: "secret123"},
			seed:    alice,
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:    "wrong password returns invalid credentials",
			input:   LoginInput{Email: "alice@example.com", Password: "wrong-password"},
			seed:    alice,
			compare: boolp(false),
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:    "repository error propagated",
			input:   LoginInput{Email: "alice@example.com", Password: "secret123"},
			seed:    alice,
			repoErr: errDB,
			wantErr: errDB,
		},
		{
			name:    "token generation error propagated",
			input:   LoginInput{Email: "alice@example.com", Password: "secret123"},
			seed:    alice,
			compare: boolp(true),
			genErr:  errJWT,
			wantErr: errJWT,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, hasher, tokens := newTestService(t)
			if tt.seed != nil {
				if err := repo.Create(context.Background(), tt.seed); err != nil {
					t.Fatal(err)
				}
			}
			repo.err = tt.repoErr

			if tt.compare != nil {
				hasher.EXPECT().Compare(hashedSecret, tt.input.Password).Return(*tt.compare)
			}
			if tt.wantToken != "" && tt.genErr == nil {
				tokens.EXPECT().Generate(tt.seed.ID).Return(tt.wantToken, nil)
			}
			if tt.genErr != nil {
				tokens.EXPECT().Generate(tt.seed.ID).Return("", tt.genErr)
			}

			out, err := svc.Login(context.Background(), tt.input)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Login() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Login() error = %v", err)
			}
			if out.Token != tt.wantToken {
				t.Errorf("Token = %q, want %q", out.Token, tt.wantToken)
			}
			if out.User.ID != tt.seed.ID {
				t.Errorf("User.ID = %q, want %q", out.User.ID, tt.seed.ID)
			}
			if out.User.Email != "alice@example.com" {
				t.Errorf("User.Email = %q, want alice@example.com", out.User.Email)
			}
		})
	}
}

// ---- GetByID ----

func TestGetByID(t *testing.T) {
	tests := []struct {
		name    string
		seed    *domain.User
		repoErr error
		id      string
		wantErr error
	}{
		{name: "success", id: "alice", seed: &domain.User{ID: "alice", Name: "Alice", Email: "alice@example.com", Password: hashedSecret}},
		{name: "empty id", id: "", wantErr: domain.ErrInvalidID},
		{name: "not found", id: "missing", wantErr: domain.ErrNotFound},
		{name: "repository error", id: "alice", seed: &domain.User{ID: "alice", Name: "Alice"}, repoErr: errDB, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _ := newTestService(t)
			if tt.seed != nil {
				if err := repo.Create(context.Background(), tt.seed); err != nil {
					t.Fatal(err)
				}
			}
			repo.err = tt.repoErr

			out, err := svc.GetByID(context.Background(), tt.id)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetByID() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}
			if out.ID != tt.id {
				t.Errorf("GetByID() = %+v, want id %q", out, tt.id)
			}
		})
	}
}

// ---- List ----

func TestList(t *testing.T) {
	tests := []struct {
		name    string
		seed    []*domain.User
		repoErr error
		wantLen int
		wantErr error
	}{
		{
			name: "success returns all users",
			seed: []*domain.User{
				{ID: "alice", Name: "Alice", Email: "alice@example.com", Password: hashedSecret},
				{ID: "bob", Name: "Bob", Email: "bob@example.com", Password: hashedSecret},
			},
			wantLen: 2,
		},
		{name: "empty returns empty list", wantLen: 0},
		{name: "repository error", repoErr: errDB, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _ := newTestService(t)
			for _, u := range tt.seed {
				if err := repo.Create(context.Background(), u); err != nil {
					t.Fatal(err)
				}
			}
			repo.err = tt.repoErr

			out, err := svc.List(context.Background())

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("List() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if len(out) != tt.wantLen {
				t.Errorf("len(List()) = %d, want %d", len(out), tt.wantLen)
			}
			if tt.wantLen == 0 && out == nil {
				t.Error("List() returned nil, want empty slice")
			}
		})
	}
}

// ---- Update ----

func newAlice() *domain.User {
	return &domain.User{ID: "alice", Name: "Alice", Email: "alice@example.com", Password: hashedSecret}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name        string
		seed        []*domain.User
		repoErr     error
		findByEmail error
		updateErr   error
		id          string
		input       UpdateUserInput
		wantName    string
		wantEmail   string
		wantField   string
		wantErr     error
	}{
		{
			name:      "success updates name only",
			seed:      []*domain.User{newAlice()},
			id:        "alice",
			input:     UpdateUserInput{Name: strp("Alice Cooper")},
			wantName:  "Alice Cooper",
			wantEmail: "alice@example.com",
		},
		{
			name:      "success updates email with normalization",
			seed:      []*domain.User{newAlice()},
			id:        "alice",
			input:     UpdateUserInput{Email: strp("  NEW@Example.COM ")},
			wantName:  "Alice",
			wantEmail: "new@example.com",
		},
		{
			name:      "success updates name and email",
			seed:      []*domain.User{newAlice()},
			id:        "alice",
			input:     UpdateUserInput{Name: strp("Alice Cooper"), Email: strp("alice.cooper@example.com")},
			wantName:  "Alice Cooper",
			wantEmail: "alice.cooper@example.com",
		},
		{
			name:      "same email allowed",
			seed:      []*domain.User{newAlice()},
			id:        "alice",
			input:     UpdateUserInput{Email: strp("ALICE@example.com")},
			wantName:  "Alice",
			wantEmail: "alice@example.com",
		},
		{
			name:      "no fields provided rejected",
			seed:      []*domain.User{newAlice()},
			id:        "alice",
			input:     UpdateUserInput{},
			wantField: "body",
		},
		{
			name:    "empty id rejected",
			id:      "",
			input:   UpdateUserInput{Name: strp("X")},
			wantErr: domain.ErrInvalidID,
		},
		{
			name:      "empty name rejected",
			seed:      []*domain.User{newAlice()},
			id:        "alice",
			input:     UpdateUserInput{Name: strp("   ")},
			wantField: "name",
		},
		{
			name:      "invalid email rejected",
			seed:      []*domain.User{newAlice()},
			id:        "alice",
			input:     UpdateUserInput{Email: strp("not-an-email")},
			wantField: "email",
		},
		{
			name:    "duplicate email rejected",
			seed:    []*domain.User{newAlice(), {ID: "bob", Name: "Bob", Email: "bob@example.com", Password: hashedSecret}},
			id:      "alice",
			input:   UpdateUserInput{Email: strp("bob@example.com")},
			wantErr: domain.ErrDuplicateEmail,
		},
		{
			name:    "not found",
			seed:    []*domain.User{newAlice()},
			id:      "missing",
			input:   UpdateUserInput{Name: strp("X")},
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "find by id error propagated",
			seed:    []*domain.User{newAlice()},
			id:      "alice",
			repoErr: errDB,
			input:   UpdateUserInput{Name: strp("X")},
			wantErr: errDB,
		},
		{
			name:        "duplicate check error propagated",
			seed:        []*domain.User{newAlice()},
			id:          "alice",
			findByEmail: errDB,
			input:       UpdateUserInput{Email: strp("new@example.com")},
			wantErr:     errDB,
		},
		{
			name:      "repository update error propagated",
			seed:      []*domain.User{newAlice()},
			id:        "alice",
			updateErr: errCreate,
			input:     UpdateUserInput{Name: strp("Alice Cooper")},
			wantErr:   errCreate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _ := newTestService(t)
			for _, u := range tt.seed {
				if err := repo.Create(context.Background(), u); err != nil {
					t.Fatal(err)
				}
			}
			repo.err = tt.repoErr
			repo.findByEmailErr = tt.findByEmail
			repo.updateErr = tt.updateErr

			out, err := svc.Update(context.Background(), tt.id, tt.input)

			if tt.wantField != "" {
				wantValidationErr(t, tt.wantField, err)
				return
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}
			if out.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", out.Name, tt.wantName)
			}
			if out.Email != tt.wantEmail {
				t.Errorf("Email = %q, want %q", out.Email, tt.wantEmail)
			}
		})
	}
}

// ---- Delete ----

func TestDelete(t *testing.T) {
	tests := []struct {
		name    string
		seed    *domain.User
		repoErr error
		id      string
		wantErr error
	}{
		{name: "success", id: "alice", seed: &domain.User{ID: "alice", Name: "Alice", Email: "alice@example.com", Password: hashedSecret}},
		{name: "empty id", id: "", wantErr: domain.ErrInvalidID},
		{name: "not found", id: "missing", wantErr: domain.ErrNotFound},
		{name: "repository error", id: "alice", seed: &domain.User{ID: "alice", Name: "Alice"}, repoErr: errDB, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _ := newTestService(t)
			if tt.seed != nil {
				if err := repo.Create(context.Background(), tt.seed); err != nil {
					t.Fatal(err)
				}
			}
			repo.err = tt.repoErr

			err := svc.Delete(context.Background(), tt.id)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Delete() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if _, err := repo.FindByID(context.Background(), tt.id); !errors.Is(err, domain.ErrNotFound) {
				t.Errorf("user still exists after Delete(): %v", err)
			}
		})
	}
}

func strp(s string) *string { return &s }

// ---- Search ----

func TestSearch(t *testing.T) {
	seed := func(t *testing.T, repo *memoryRepo) {
		t.Helper()
		for _, u := range []*domain.User{
			{ID: "1", Name: "Alice Johnson", Email: "alice@example.com", Password: hashedSecret},
			{ID: "2", Name: "Bob Smith", Email: "bob@corp.com", Password: hashedSecret},
			{ID: "3", Name: "Alicia Keys", Email: "keys@music.io", Password: hashedSecret},
			{ID: "4", Name: "Carol", Email: "alice2@example.com", Password: hashedSecret},
		} {
			if err := repo.Create(context.Background(), u); err != nil {
				t.Fatal(err)
			}
		}
	}

	t.Run("prefix matches name and email, name ranked higher", func(t *testing.T) {
		svc, repo, _, _ := newTestService(t)
		seed(t, repo)

		out, err := svc.Search(context.Background(), SearchUsersInput{Query: "ali"})
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(out) != 3 {
			t.Fatalf("got %d results, want 3", len(out))
		}
		// Alice (name exact-prefix) and Alicia (name prefix) outrank Carol (email).
		if out[0].Rank < out[2].Rank {
			t.Errorf("expected name matches ranked above email match: %+v", out)
		}
		if out[len(out)-1].User.ID != "4" {
			t.Errorf("email-only match should rank last, got %q", out[len(out)-1].User.ID)
		}
	})

	t.Run("quoted token requires exact word", func(t *testing.T) {
		svc, repo, _, _ := newTestService(t)
		seed(t, repo)

		out, err := svc.Search(context.Background(), SearchUsersInput{Query: `"alice"`})
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		// Only alice@example.com matches the whole word "alice"; alice2@ and
		// the name-prefix matches (Alice/Alicia) do not.
		if len(out) != 1 || out[0].User.ID != "1" {
			t.Fatalf("got %+v, want single exact-word match alice@example.com", out)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		svc, repo, _, _ := newTestService(t)
		seed(t, repo)

		out, err := svc.Search(context.Background(), SearchUsersInput{Query: "BOB"})
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(out) != 1 || out[0].User.ID != "2" {
			t.Fatalf("got %+v, want single Bob match", out)
		}
	})

	t.Run("empty query returns empty slice", func(t *testing.T) {
		svc, repo, _, _ := newTestService(t)
		seed(t, repo)

		out, err := svc.Search(context.Background(), SearchUsersInput{Query: "   "})
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if out == nil || len(out) != 0 {
			t.Fatalf("got %+v, want non-nil empty slice", out)
		}
	})

	t.Run("repository error propagated", func(t *testing.T) {
		svc, repo, _, _ := newTestService(t)
		repo.err = errDB

		if _, err := svc.Search(context.Background(), SearchUsersInput{Query: "ali"}); !errors.Is(err, errDB) {
			t.Fatalf("Search() error = %v, want %v", err, errDB)
		}
	})
}
