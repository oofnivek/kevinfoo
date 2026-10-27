// Package push stores Web Push subscriptions and sends notifications to them.
package push

import (
	"context"
	"time"
)

// Subscription is a browser's Web Push subscription, as returned by
// PushManager.subscribe(), registered under a user ID and topic. One browser
// (endpoint) can be registered under many user ID + topic pairs.
type Subscription struct {
	Endpoint  string
	UserID    string
	Topic     string
	P256dh    string
	Auth      string
	CreatedAt time.Time
	// LastUsedAt is when the user ID + topic was last subscribed to or sent
	// to. Subscriptions unused for longer than Retention are deleted.
	LastUsedAt time.Time
}

// Retention is how long a subscription is kept without being used.
const Retention = 90 * 24 * time.Hour

type Repository interface {
	// Save inserts s, or updates its keys if the endpoint is already
	// registered under the same user ID and topic. Either way it marks the
	// subscription as used now.
	Save(ctx context.Context, s Subscription) error
	// Touch marks every subscription under userID + topic as used now.
	Touch(ctx context.Context, userID, topic string) error
	// ListByTopic returns every subscription registered under userID + topic.
	ListByTopic(ctx context.Context, userID, topic string) ([]Subscription, error)
	// ListByEndpoint returns every user ID + topic a browser is registered under.
	ListByEndpoint(ctx context.Context, endpoint string) ([]Subscription, error)
	// Delete removes one registration. Deleting an unknown one is not an error.
	Delete(ctx context.Context, endpoint, userID, topic string) error
	// DeleteEndpoint removes every registration for a browser.
	DeleteEndpoint(ctx context.Context, endpoint string) error
	// DeleteUnused removes subscriptions not used within olderThan,
	// returning how many were removed.
	DeleteUnused(ctx context.Context, olderThan time.Duration) (int64, error)
}
