package push

import (
	"context"
	"database/sql"
	"fmt"
)

type sqliteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository returns a Repository backed by SQLite.
func NewSQLiteRepository(db *sql.DB) Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Save(ctx context.Context, s Subscription) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO push_subscriptions (endpoint, user_id, topic, p256dh, auth)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (endpoint, user_id, topic) DO UPDATE SET p256dh = excluded.p256dh, auth = excluded.auth`,
		s.Endpoint, s.UserID, s.Topic, s.P256dh, s.Auth)
	if err != nil {
		return fmt.Errorf("save push subscription: %w", err)
	}
	return nil
}

func (r *sqliteRepository) ListByTopic(ctx context.Context, userID, topic string) ([]Subscription, error) {
	return r.list(ctx, "WHERE user_id = ? AND topic = ?", userID, topic)
}

func (r *sqliteRepository) ListByEndpoint(ctx context.Context, endpoint string) ([]Subscription, error) {
	return r.list(ctx, "WHERE endpoint = ?", endpoint)
}

func (r *sqliteRepository) list(ctx context.Context, where string, args ...any) ([]Subscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT endpoint, user_id, topic, p256dh, auth, created_at
		FROM push_subscriptions
		`+where+`
		ORDER BY created_at`, args...)
	if err != nil {
		return nil, fmt.Errorf("list push subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.Endpoint, &s.UserID, &s.Topic, &s.P256dh, &s.Auth, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan push subscription: %w", err)
		}
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list push subscriptions: %w", err)
	}

	return subs, nil
}

func (r *sqliteRepository) Delete(ctx context.Context, endpoint, userID, topic string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM push_subscriptions WHERE endpoint = ? AND user_id = ? AND topic = ?",
		endpoint, userID, topic)
	if err != nil {
		return fmt.Errorf("delete push subscription: %w", err)
	}
	return nil
}

func (r *sqliteRepository) DeleteEndpoint(ctx context.Context, endpoint string) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM push_subscriptions WHERE endpoint = ?", endpoint); err != nil {
		return fmt.Errorf("delete push subscriptions: %w", err)
	}
	return nil
}
