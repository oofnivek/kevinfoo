package loginlog

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// retention is how long login attempts are kept before the TTL index
// expires them.
const retention = 90 * 24 * time.Hour

type mongoLogger struct {
	coll *mongo.Collection
}

// NewMongoLogger returns a Logger backed by MongoDB. The login_attempts
// collection is created lazily by MongoDB on first write; this sets up the
// supporting indexes: a TTL index that expires attempts after 90 days, and
// an index on username to look up recent attempts for a given account.
func NewMongoLogger(ctx context.Context, db *mongo.Database) (Logger, error) {
	coll := db.Collection("login_attempts")

	_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "created_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(int32(retention.Seconds())),
		},
		{
			Keys: bson.D{{Key: "username", Value: 1}, {Key: "created_at", Value: -1}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create login_attempts indexes: %w", err)
	}

	return &mongoLogger{coll: coll}, nil
}

type mongoAttempt struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Username  string        `bson:"username"`
	IP        string        `bson:"ip"`
	UserAgent string        `bson:"user_agent"`
	Success   bool          `bson:"success"`
	Reason    string        `bson:"reason,omitempty"`
	CreatedAt time.Time     `bson:"created_at"`
}

func (m *mongoLogger) Log(ctx context.Context, a Attempt) error {
	doc := mongoAttempt{
		ID:        bson.NewObjectID(),
		Username:  a.Username,
		IP:        a.IP,
		UserAgent: a.UserAgent,
		Success:   a.Success,
		Reason:    a.Reason,
		CreatedAt: a.CreatedAt,
	}

	if _, err := m.coll.InsertOne(ctx, doc); err != nil {
		return fmt.Errorf("log login attempt: %w", err)
	}

	return nil
}
