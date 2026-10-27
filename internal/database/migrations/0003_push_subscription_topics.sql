-- Subscriptions are now registered per user ID + topic. Existing rows have
-- neither, so they are dropped; browsers re-register from the page.
DROP TABLE push_subscriptions;

CREATE TABLE push_subscriptions (
    endpoint   TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    topic      TEXT NOT NULL,
    p256dh     TEXT NOT NULL,
    auth       TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (endpoint, user_id, topic)
);

CREATE INDEX idx_push_subscriptions_user_topic ON push_subscriptions (user_id, topic);
