package push

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoRepository struct {
	coll *mongo.Collection
}

// NewMongoRepository returns a Repository backed by MongoDB, storing
// subscriptions in db's "subscriptions" collection. The collection and
// database are created lazily by MongoDB on first write; this sets up the
// unique index Save upserts against, an index for looking up a topic's
// subscribers, and a TTL index that has MongoDB delete subscriptions unused
// for Retention.
func NewMongoRepository(ctx context.Context, db *mongo.Database) (Repository, error) {
	coll := db.Collection("subscriptions")

	// Documents without last_used_at would never expire, so give any from
	// before it existed a full retention period from now.
	_, err := coll.UpdateMany(ctx,
		bson.M{"last_used_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"last_used_at": time.Now().UTC()}})
	if err != nil {
		return nil, fmt.Errorf("backfill push subscriptions last_used_at: %w", err)
	}

	_, err = coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "endpoint", Value: 1}, {Key: "user_id", Value: 1}, {Key: "topic", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "topic", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "last_used_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(int32(Retention.Seconds())),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create push subscriptions indexes: %w", err)
	}

	return &mongoRepository{coll: coll}, nil
}

type mongoSubscription struct {
	Endpoint   string    `bson:"endpoint"`
	UserID     string    `bson:"user_id"`
	Topic      string    `bson:"topic"`
	P256dh     string    `bson:"p256dh"`
	Auth       string    `bson:"auth"`
	CreatedAt  time.Time `bson:"created_at"`
	LastUsedAt time.Time `bson:"last_used_at"`
}

func (r *mongoRepository) Save(ctx context.Context, s Subscription) error {
	now := time.Now().UTC()
	filter := bson.M{"endpoint": s.Endpoint, "user_id": s.UserID, "topic": s.Topic}
	update := bson.M{
		"$set":         bson.M{"p256dh": s.P256dh, "auth": s.Auth, "last_used_at": now},
		"$setOnInsert": bson.M{"created_at": now},
	}

	_, err := r.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("save push subscription: %w", err)
	}
	return nil
}

func (r *mongoRepository) Touch(ctx context.Context, userID, topic string) error {
	_, err := r.coll.UpdateMany(ctx,
		bson.M{"user_id": userID, "topic": topic},
		bson.M{"$set": bson.M{"last_used_at": time.Now().UTC()}})
	if err != nil {
		return fmt.Errorf("touch push subscriptions: %w", err)
	}
	return nil
}

func (r *mongoRepository) ListByTopic(ctx context.Context, userID, topic string) ([]Subscription, error) {
	return r.list(ctx, bson.M{"user_id": userID, "topic": topic})
}

func (r *mongoRepository) ListByEndpoint(ctx context.Context, endpoint string) ([]Subscription, error) {
	return r.list(ctx, bson.M{"endpoint": endpoint})
}

func (r *mongoRepository) list(ctx context.Context, filter bson.M) ([]Subscription, error) {
	cur, err := r.coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("list push subscriptions: %w", err)
	}
	defer cur.Close(ctx)

	var subs []Subscription
	for cur.Next(ctx) {
		var m mongoSubscription
		if err := cur.Decode(&m); err != nil {
			return nil, fmt.Errorf("decode push subscription: %w", err)
		}
		subs = append(subs, Subscription{
			Endpoint:   m.Endpoint,
			UserID:     m.UserID,
			Topic:      m.Topic,
			P256dh:     m.P256dh,
			Auth:       m.Auth,
			CreatedAt:  m.CreatedAt,
			LastUsedAt: m.LastUsedAt,
		})
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("list push subscriptions: %w", err)
	}

	return subs, nil
}

func (r *mongoRepository) Delete(ctx context.Context, endpoint, userID, topic string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"endpoint": endpoint, "user_id": userID, "topic": topic})
	if err != nil {
		return fmt.Errorf("delete push subscription: %w", err)
	}
	return nil
}

func (r *mongoRepository) DeleteEndpoint(ctx context.Context, endpoint string) error {
	if _, err := r.coll.DeleteMany(ctx, bson.M{"endpoint": endpoint}); err != nil {
		return fmt.Errorf("delete push subscriptions: %w", err)
	}
	return nil
}

// DeleteUnused is a fallback for the TTL index, which MongoDB applies on
// its own schedule (about once a minute).
func (r *mongoRepository) DeleteUnused(ctx context.Context, olderThan time.Duration) (int64, error) {
	res, err := r.coll.DeleteMany(ctx, bson.M{"last_used_at": bson.M{"$lt": time.Now().UTC().Add(-olderThan)}})
	if err != nil {
		return 0, fmt.Errorf("delete unused push subscriptions: %w", err)
	}
	return res.DeletedCount, nil
}
