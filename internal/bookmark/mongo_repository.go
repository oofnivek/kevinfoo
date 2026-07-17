package bookmark

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoRepository struct {
	coll *mongo.Collection
}

// NewMongoRepository returns a Repository backed by MongoDB. The bookmarks
// collection and database are created lazily by MongoDB itself on first
// write; the only setup needed here is the supporting index.
func NewMongoRepository(ctx context.Context, db *mongo.Database) (Repository, error) {
	coll := db.Collection("bookmarks")

	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	})
	if err != nil {
		return nil, fmt.Errorf("create bookmarks index: %w", err)
	}

	return &mongoRepository{coll: coll}, nil
}

type mongoBookmark struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	Title       string        `bson:"title"`
	URL         string        `bson:"url"`
	Description string        `bson:"description"`
	Tags        string        `bson:"tags"`
	CreatedAt   time.Time     `bson:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at"`
}

func (m mongoBookmark) toBookmark() Bookmark {
	return Bookmark{
		ID:          m.ID.Hex(),
		Title:       m.Title,
		URL:         m.URL,
		Description: m.Description,
		Tags:        m.Tags,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func (r *mongoRepository) List(ctx context.Context, query string) ([]Bookmark, error) {
	filter := bson.M{}
	if query != "" {
		re := bson.Regex{Pattern: query, Options: "i"}
		filter = bson.M{"$or": bson.A{
			bson.M{"title": re},
			bson.M{"url": re},
			bson.M{"description": re},
			bson.M{"tags": re},
		}}
	}

	cur, err := r.coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}
	defer cur.Close(ctx)

	var bookmarks []Bookmark
	for cur.Next(ctx) {
		var m mongoBookmark
		if err := cur.Decode(&m); err != nil {
			return nil, fmt.Errorf("decode bookmark: %w", err)
		}
		bookmarks = append(bookmarks, m.toBookmark())
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}

	return bookmarks, nil
}

func (r *mongoRepository) GetByID(ctx context.Context, id string) (Bookmark, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return Bookmark{}, ErrNotFound
	}

	var m mongoBookmark
	err = r.coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Bookmark{}, ErrNotFound
	}
	if err != nil {
		return Bookmark{}, fmt.Errorf("get bookmark %s: %w", id, err)
	}

	return m.toBookmark(), nil
}

func (r *mongoRepository) Create(ctx context.Context, b Bookmark) (Bookmark, error) {
	now := time.Now().UTC()
	m := mongoBookmark{
		ID:          bson.NewObjectID(),
		Title:       b.Title,
		URL:         b.URL,
		Description: b.Description,
		Tags:        b.Tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := r.coll.InsertOne(ctx, m); err != nil {
		return Bookmark{}, fmt.Errorf("create bookmark: %w", err)
	}

	return m.toBookmark(), nil
}

func (r *mongoRepository) Update(ctx context.Context, b Bookmark) (Bookmark, error) {
	oid, err := bson.ObjectIDFromHex(b.ID)
	if err != nil {
		return Bookmark{}, ErrNotFound
	}

	update := bson.M{"$set": bson.M{
		"title":       b.Title,
		"url":         b.URL,
		"description": b.Description,
		"tags":        b.Tags,
		"updated_at":  time.Now().UTC(),
	}}

	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return Bookmark{}, fmt.Errorf("update bookmark %s: %w", b.ID, err)
	}
	if res.MatchedCount == 0 {
		return Bookmark{}, ErrNotFound
	}

	return r.GetByID(ctx, b.ID)
}

func (r *mongoRepository) Delete(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrNotFound
	}

	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return fmt.Errorf("delete bookmark %s: %w", id, err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}

	return nil
}
