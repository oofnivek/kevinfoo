CREATE TABLE bookmarks (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    url         TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    tags        TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bookmarks_created_at ON bookmarks (created_at DESC);
