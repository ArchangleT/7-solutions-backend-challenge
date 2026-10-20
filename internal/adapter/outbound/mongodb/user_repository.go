package mongodb

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	domain "github.com/ArchangleT/7-solutions-backend-challenge/internal/domain/user"
)

const usersCollection = "users"

// userDoc is the MongoDB representation of a user. The adapter owns this
// mapping so the domain entity stays free of driver types.
type userDoc struct {
	ID        bson.ObjectID `bson:"_id"`
	Name      string        `bson:"name"`
	Email     string        `bson:"email"`
	Password  string        `bson:"password"`
	CreatedAt time.Time     `bson:"created_at"`
}

// userCollection is the subset of *mongo.Collection that UserRepository uses,
// declared as an interface so unit tests can drive the logic paths with a fake.
type userCollection interface {
	InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error)
	FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult
	Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error)
	ReplaceOne(ctx context.Context, filter any, replacement any, opts ...options.Lister[options.ReplaceOptions]) (*mongo.UpdateResult, error)
	DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error)
	EstimatedDocumentCount(ctx context.Context, opts ...options.Lister[options.EstimatedDocumentCountOptions]) (int64, error)
	Indexes() mongo.IndexView
}

// UserRepository implements user.Repository backed by MongoDB.
type UserRepository struct {
	collection userCollection
}

func NewUserRepository(collection *mongo.Collection) *UserRepository {
	return &UserRepository{collection: collection}
}

// EnsureIndexes creates the unique index on email.
func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create unique email index: %w", err)
	}
	return nil
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	if u.ID == "" {
		u.ID = bson.NewObjectID().Hex()
	}
	oid, err := bson.ObjectIDFromHex(u.ID)
	if err != nil {
		return domain.ErrInvalidID
	}

	_, err = r.collection.InsertOne(ctx, userDoc{
		ID:        oid,
		Name:      u.Name,
		Email:     u.Email,
		Password:  u.Password,
		CreatedAt: u.CreatedAt,
	})
	if mongo.IsDuplicateKeyError(err) {
		return domain.ErrDuplicateEmail
	}
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var doc userDoc
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return toDomain(&doc), nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var doc userDoc
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return toDomain(&doc), nil
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	cursor, err := r.collection.Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer cursor.Close(ctx)

	users := make([]domain.User, 0)
	for cursor.Next(ctx) {
		var doc userDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode user: %w", err)
		}
		users = append(users, *toDomain(&doc))
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("list users cursor: %w", err)
	}
	return users, nil
}

// Search matches users whose name or email contains a word starting with any
// of the query tokens (word-prefix, case-insensitive), mirroring Apple Notes.
// MongoDB has no first-class word-prefix operator, so each token is anchored
// at a word boundary via a regex, then results are sorted "Top Hits" first:
// name matches outrank email matches and ties break toward the most recent.
func (r *UserRepository) Search(ctx context.Context, query domain.SearchQuery) ([]domain.User, error) {
	filter := buildSearchFilter(query)
	if filter == nil {
		return []domain.User{}, nil
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer cursor.Close(ctx)

	users := make([]domain.User, 0)
	for cursor.Next(ctx) {
		var doc userDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode user: %w", err)
		}
		users = append(users, *toDomain(&doc))
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("search users cursor: %w", err)
	}

	sortTopHits(users, query)
	return users, nil
}

// buildSearchFilter builds an $or over (name, email) for every prefix token
// and exact token. Returns nil when there are no tokens to match.
func buildSearchFilter(query domain.SearchQuery) bson.M {
	var clauses []bson.M

	for _, token := range query.Tokens {
		expr := wordPrefixRegex(regexp.QuoteMeta(token))
		clauses = append(clauses,
			bson.M{"name": bson.M{"$regex": expr, "$options": "i"}},
			bson.M{"email": bson.M{"$regex": expr, "$options": "i"}},
		)
	}
	for _, token := range query.ExactTokens {
		expr := `(^|[^a-z0-9])` + regexp.QuoteMeta(token) + `([^a-z0-9]|$)`
		clauses = append(clauses,
			bson.M{"name": bson.M{"$regex": expr, "$options": "i"}},
			bson.M{"email": bson.M{"$regex": expr, "$options": "i"}},
		)
	}

	if len(clauses) == 0 {
		return nil
	}
	return bson.M{"$or": clauses}
}

// wordPrefixRegex builds a regex that matches token as a whole word prefix:
// either at the start of the string, or preceded by a non-alphanumeric
// separator. It approximates word tokenization without a text index.
func wordPrefixRegex(token string) string {
	return "^" + token + `|(^|[^a-zA-Z0-9])` + token
}

// sortTopHits orders users the way Apple Notes surfaces "Top Hits": stronger
// field matches first (name over email), then most recently created. It reuses
// the domain scoring so the ordering and the API's reported rank agree.
func sortTopHits(users []domain.User, query domain.SearchQuery) {
	sort.SliceStable(users, func(i, j int) bool {
		si, _ := query.Score(&users[i])
		sj, _ := query.Score(&users[j])
		if si != sj {
			return si > sj
		}
		return users[i].CreatedAt.After(users[j].CreatedAt)
	})
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	oid, err := bson.ObjectIDFromHex(u.ID)
	if err != nil {
		return domain.ErrInvalidID
	}

	res, err := r.collection.ReplaceOne(ctx, bson.M{"_id": oid}, userDoc{
		ID:        oid,
		Name:      u.Name,
		Email:     u.Email,
		Password:  u.Password,
		CreatedAt: u.CreatedAt,
	})
	if mongo.IsDuplicateKeyError(err) {
		return domain.ErrDuplicateEmail
	}
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrInvalidID
	}

	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	n, err := r.collection.EstimatedDocumentCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

func toDomain(d *userDoc) *domain.User {
	return &domain.User{
		ID:        d.ID.Hex(),
		Name:      d.Name,
		Email:     d.Email,
		Password:  d.Password,
		CreatedAt: d.CreatedAt,
	}
}
