package mongodb

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	domain "github.com/ArchangleT/7-solutions-backend-challenge/internal/domain/user"
)

var errMongo = errors.New("mongo failure")

// fakeCollection implements userCollection so the repo logic paths can be
// driven without a real MongoDB server.
type fakeCollection struct {
	insertOneResult *mongo.InsertOneResult
	insertOneErr    error

	findOneDoc any
	findOneErr error

	findDocs      []any
	findErr       error
	findCursorErr error

	replaceOneResult *mongo.UpdateResult
	replaceOneErr    error

	deleteOneResult *mongo.DeleteResult
	deleteOneErr    error

	count    int64
	countErr error
}

func (f *fakeCollection) InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	return f.insertOneResult, f.insertOneErr
}

func (f *fakeCollection) FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult {
	return mongo.NewSingleResultFromDocument(f.findOneDoc, f.findOneErr, nil)
}

func (f *fakeCollection) Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return mongo.NewCursorFromDocuments(f.findDocs, f.findCursorErr, nil)
}

func (f *fakeCollection) ReplaceOne(ctx context.Context, filter any, replacement any, opts ...options.Lister[options.ReplaceOptions]) (*mongo.UpdateResult, error) {
	return f.replaceOneResult, f.replaceOneErr
}

func (f *fakeCollection) DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {
	return f.deleteOneResult, f.deleteOneErr
}

func (f *fakeCollection) EstimatedDocumentCount(ctx context.Context, opts ...options.Lister[options.EstimatedDocumentCountOptions]) (int64, error) {
	return f.count, f.countErr
}

func (f *fakeCollection) Indexes() mongo.IndexView {
	return mongo.IndexView{}
}

func sampleUserDoc(hexID, email string) userDoc {
	oid, err := bson.ObjectIDFromHex(hexID)
	if err != nil {
		panic(err)
	}
	return userDoc{ID: oid, Name: "Alice", Email: email, Password: "hash", CreatedAt: time.Now().UTC()}
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name string
		user *domain.User
		col  *fakeCollection
		want error
	}{
		{
			name: "success assigns id when empty",
			user: &domain.User{Name: "Alice", Email: "alice@example.com", Password: "hash", CreatedAt: time.Now().UTC()},
			col:  &fakeCollection{insertOneResult: &mongo.InsertOneResult{}},
		},
		{
			name: "duplicate key maps to ErrDuplicateEmail",
			user: &domain.User{ID: "0123456789abcdef01234567", Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now().UTC()},
			col:  &fakeCollection{insertOneErr: mongo.CommandError{Code: 11000}},
			want: domain.ErrDuplicateEmail,
		},
		{
			name: "driver error is wrapped",
			user: &domain.User{Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now().UTC()},
			col:  &fakeCollection{insertOneErr: errMongo},
			want: errMongo,
		},
		{
			name: "invalid hex maps to ErrInvalidID",
			user: &domain.User{ID: "not-a-hex-id", Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now().UTC()},
			col:  &fakeCollection{},
			want: domain.ErrInvalidID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &UserRepository{collection: tt.col}
			err := repo.Create(context.Background(), tt.user)
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Create() error = %v, want nil", err)
				}
				if tt.user.ID == "" {
					t.Error("Create() did not assign an ID")
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("Create() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestFindByID(t *testing.T) {
	const hexID = "0123456789abcdef01234567"
	tests := []struct {
		name      string
		id        string
		col       *fakeCollection
		wantEmail string
		wantErr   error
	}{
		{
			name:      "success",
			id:        hexID,
			col:       &fakeCollection{findOneDoc: sampleUserDoc(hexID, "alice@example.com")},
			wantEmail: "alice@example.com",
		},
		{
			name:    "no document maps to ErrNotFound",
			id:      hexID,
			col:     &fakeCollection{findOneDoc: bson.M{}, findOneErr: mongo.ErrNoDocuments},
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "invalid hex maps to ErrNotFound",
			id:      "not-a-hex-id",
			col:     &fakeCollection{},
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "driver error is wrapped",
			id:      hexID,
			col:     &fakeCollection{findOneDoc: bson.M{}, findOneErr: errMongo},
			wantErr: errMongo,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &UserRepository{collection: tt.col}
			got, err := repo.FindByID(context.Background(), tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("FindByID() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("FindByID() error = %v, want nil", err)
			}
			if got.Email != tt.wantEmail {
				t.Errorf("FindByID() email = %q, want %q", got.Email, tt.wantEmail)
			}
		})
	}
}

func TestFindByEmail(t *testing.T) {
	const hexID = "0123456789abcdef01234567"
	tests := []struct {
		name      string
		col       *fakeCollection
		wantEmail string
		wantErr   error
	}{
		{
			name:      "success",
			col:       &fakeCollection{findOneDoc: sampleUserDoc(hexID, "alice@example.com")},
			wantEmail: "alice@example.com",
		},
		{
			name:    "no document maps to ErrNotFound",
			col:     &fakeCollection{findOneDoc: bson.M{}, findOneErr: mongo.ErrNoDocuments},
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "driver error is wrapped",
			col:     &fakeCollection{findOneDoc: bson.M{}, findOneErr: errMongo},
			wantErr: errMongo,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &UserRepository{collection: tt.col}
			got, err := repo.FindByEmail(context.Background(), "alice@example.com")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("FindByEmail() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("FindByEmail() error = %v, want nil", err)
			}
			if got.Email != tt.wantEmail {
				t.Errorf("FindByEmail() email = %q, want %q", got.Email, tt.wantEmail)
			}
		})
	}
}

func TestList(t *testing.T) {
	const hexA = "0123456789abcdef01234567"
	const hexB = "0123456789abcdef01234568"
	tests := []struct {
		name      string
		col       *fakeCollection
		wantCount int
		wantErr   error
	}{
		{
			name: "success returns documents",
			col: &fakeCollection{findDocs: []any{
				sampleUserDoc(hexA, "alice@example.com"),
				sampleUserDoc(hexB, "bob@example.com"),
			}},
			wantCount: 2,
		},
		{
			name:      "empty collection returns empty slice",
			col:       &fakeCollection{findDocs: nil},
			wantCount: 0,
		},
		{
			name:    "find error is wrapped",
			col:     &fakeCollection{findErr: errMongo},
			wantErr: errMongo,
		},
		{
			name:    "cursor error is wrapped",
			col:     &fakeCollection{findDocs: []any{sampleUserDoc(hexA, "alice@example.com")}, findCursorErr: errMongo},
			wantErr: errMongo,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &UserRepository{collection: tt.col}
			got, err := repo.List(context.Background())
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("List() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("List() error = %v, want nil", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("List() returned %d users, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestList_DecodeErrorIsWrapped(t *testing.T) {
	repo := &UserRepository{collection: &fakeCollection{findDocs: []any{bson.M{"email": 42}}}}
	_, err := repo.List(context.Background())
	if err == nil || !strings.Contains(err.Error(), "decode user") {
		t.Fatalf("List() error = %v, want decode error to be wrapped", err)
	}
}

func TestSearch(t *testing.T) {
	const hexA = "0123456789abcdef01234567"
	const hexB = "0123456789abcdef01234568"
	const hexC = "0123456789abcdef01234569"

	alice := userDoc{ID: mustOID(hexA), Name: "Alice Johnson", Email: "alice@example.com", CreatedAt: time.Unix(100, 0).UTC()}
	alicia := userDoc{ID: mustOID(hexB), Name: "Alicia Keys", Email: "keys@music.io", CreatedAt: time.Unix(200, 0).UTC()}
	carol := userDoc{ID: mustOID(hexC), Name: "Carol", Email: "alice2@example.com", CreatedAt: time.Unix(300, 0).UTC()}

	t.Run("empty query returns empty slice", func(t *testing.T) {
		repo := &UserRepository{collection: &fakeCollection{findDocs: []any{alice}}}
		got, err := repo.Search(context.Background(), domain.SearchQuery{})
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("got %+v, want non-nil empty slice", got)
		}
	})

	t.Run("name matches outrank email matches", func(t *testing.T) {
		repo := &UserRepository{collection: &fakeCollection{findDocs: []any{carol, alicia, alice}}}
		got, err := repo.Search(context.Background(), domain.ParseSearchQuery("ali"))
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d users, want 3", len(got))
		}
		// Alice (name + email) > Alicia (name only) > Carol (email only).
		wantOrder := []string{hexA, hexB, hexC}
		for i, want := range wantOrder {
			if got[i].ID != want {
				t.Fatalf("order[%d] = %q, want %q (got %+v)", i, got[i].ID, want, got)
			}
		}
	})

	t.Run("ties on score break toward most recent", func(t *testing.T) {
		older := userDoc{ID: mustOID(hexA), Name: "Zoe Older", Email: "zoe@a.io", CreatedAt: time.Unix(100, 0).UTC()}
		newer := userDoc{ID: mustOID(hexB), Name: "Zoe Newer", Email: "zoe@b.io", CreatedAt: time.Unix(300, 0).UTC()}
		repo := &UserRepository{collection: &fakeCollection{findDocs: []any{older, newer}}}
		got, err := repo.Search(context.Background(), domain.ParseSearchQuery("zoe"))
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(got) != 2 || got[0].ID != hexB || got[1].ID != hexA {
			t.Fatalf("order = %+v, want newer before older", got)
		}
	})

	t.Run("find error is wrapped", func(t *testing.T) {
		repo := &UserRepository{collection: &fakeCollection{findErr: errMongo}}
		if _, err := repo.Search(context.Background(), domain.ParseSearchQuery("ali")); !errors.Is(err, errMongo) {
			t.Fatalf("Search() error = %v, want %v", err, errMongo)
		}
	})
}

func TestBuildSearchFilter(t *testing.T) {
	if got := buildSearchFilter(domain.SearchQuery{}); got != nil {
		t.Fatalf("empty query filter = %v, want nil", got)
	}

	filter := buildSearchFilter(domain.SearchQuery{Tokens: []string{"al"}, ExactTokens: []string{"bob"}})
	clauses, ok := filter["$or"].([]bson.M)
	if !ok || len(clauses) != 4 {
		t.Fatalf("filter clauses = %v, want 4 ($or over name/email for each token)", filter)
	}
}

func TestWordPrefixRegex(t *testing.T) {
	got := wordPrefixRegex("al")
	want := `^al|(^|[^a-zA-Z0-9])al`
	if got != want {
		t.Fatalf("wordPrefixRegex() = %q, want %q", got, want)
	}
}

func mustOID(hex string) bson.ObjectID {
	oid, err := bson.ObjectIDFromHex(hex)
	if err != nil {
		panic(err)
	}
	return oid
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name string
		id   string
		col  *fakeCollection
		want error
	}{
		{
			name: "success",
			id:   "0123456789abcdef01234567",
			col:  &fakeCollection{replaceOneResult: &mongo.UpdateResult{MatchedCount: 1}},
		},
		{
			name: "duplicate key maps to ErrDuplicateEmail",
			id:   "0123456789abcdef01234567",
			col:  &fakeCollection{replaceOneErr: mongo.CommandError{Code: 11000}},
			want: domain.ErrDuplicateEmail,
		},
		{
			name: "no matching document maps to ErrNotFound",
			id:   "0123456789abcdef01234567",
			col:  &fakeCollection{replaceOneResult: &mongo.UpdateResult{}},
			want: domain.ErrNotFound,
		},
		{
			name: "driver error is wrapped",
			id:   "0123456789abcdef01234567",
			col:  &fakeCollection{replaceOneErr: errMongo},
			want: errMongo,
		},
		{
			name: "invalid hex maps to ErrInvalidID",
			id:   "not-a-hex-id",
			col:  &fakeCollection{},
			want: domain.ErrInvalidID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &UserRepository{collection: tt.col}
			err := repo.Update(context.Background(), &domain.User{
				ID:        tt.id,
				Name:      "Alice",
				Email:     "alice@example.com",
				CreatedAt: time.Now().UTC(),
			})
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Update() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("Update() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name string
		id   string
		col  *fakeCollection
		want error
	}{
		{
			name: "success",
			id:   "0123456789abcdef01234567",
			col:  &fakeCollection{deleteOneResult: &mongo.DeleteResult{DeletedCount: 1}},
		},
		{
			name: "no matching document maps to ErrNotFound",
			id:   "0123456789abcdef01234567",
			col:  &fakeCollection{deleteOneResult: &mongo.DeleteResult{}},
			want: domain.ErrNotFound,
		},
		{
			name: "driver error is wrapped",
			id:   "0123456789abcdef01234567",
			col:  &fakeCollection{deleteOneErr: errMongo},
			want: errMongo,
		},
		{
			name: "invalid hex maps to ErrInvalidID",
			id:   "not-a-hex-id",
			col:  &fakeCollection{},
			want: domain.ErrInvalidID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &UserRepository{collection: tt.col}
			err := repo.Delete(context.Background(), tt.id)
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Delete() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("Delete() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCount(t *testing.T) {
	tests := []struct {
		name string
		col  *fakeCollection
		want int64
	}{{
		name: "success",
		col:  &fakeCollection{count: 5},
		want: 5,
	}, {
		name: "zero",
		col:  &fakeCollection{count: 0},
		want: 0,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &UserRepository{collection: tt.col}
			n, err := repo.Count(context.Background())
			if err != nil {
				t.Fatalf("Count() error = %v, want nil", err)
			}
			if n != tt.want {
				t.Errorf("Count() = %d, want %d", n, tt.want)
			}
		})
	}
}

func TestCount_DriverErrorIsWrapped(t *testing.T) {
	repo := &UserRepository{collection: &fakeCollection{countErr: errMongo}}
	if _, err := repo.Count(context.Background()); !errors.Is(err, errMongo) {
		t.Fatalf("Count() error = %v, want wrapped %v", err, errMongo)
	}
}
