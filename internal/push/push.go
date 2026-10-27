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
}

type Repository interface {
	// Save inserts s, or updates its keys if the endpoint is already
	// registered under the same user ID and topic.
	Save(ctx context.Context, s Subscription) error
	// ListByTopic returns every subscription registered under userID + topic.
	ListByTopic(ctx context.Context, userID, topic string) ([]Subscription, error)
	// ListByEndpoint returns every user ID + topic a browser is registered under.
	ListByEndpoint(ctx context.Context, endpoint string) ([]Subscription, error)
	// Delete removes one registration. Deleting an unknown one is not an error.
	Delete(ctx context.Context, endpoint, userID, topic string) error
	// DeleteEndpoint removes every registration for a browser.
	DeleteEndpoint(ctx context.Context, endpoint string) error
}
