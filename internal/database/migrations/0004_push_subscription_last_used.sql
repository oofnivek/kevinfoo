-- Track when each user ID + topic was last used (subscribed to or sent to),
-- so unused subscriptions can be cleaned up. SQLite can't add a NOT NULL
-- column with a CURRENT_TIMESTAMP default, so the table is rebuilt.
CREATE TABLE push_subscriptions_new (
    endpoint     TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    topic        TEXT NOT NULL,
    p256dh       TEXT NOT NULL,
    auth         TEXT NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (endpoint, user_id, topic)
);

INSERT INTO push_subscriptions_new (endpoint, user_id, topic, p256dh, auth, created_at, last_used_at)
SELECT endpoint, user_id, topic, p256dh, auth, created_at, CURRENT_TIMESTAMP FROM push_subscriptions;

DROP TABLE push_subscriptions;
ALTER TABLE push_subscriptions_new RENAME TO push_subscriptions;

CREATE INDEX idx_push_subscriptions_user_topic ON push_subscriptions (user_id, topic);
CREATE INDEX idx_push_subscriptions_last_used_at ON push_subscriptions (last_used_at);
