# Bookmarks

A small self-hosted bookmark manager. Go backend, SQLite storage, HTML
rendered server-side and enhanced with [htmx](https://htmx.org), styled with
[Tailwind CSS](https://tailwindcss.com).

## Project layout

```
cmd/server/            application entrypoint
internal/config/       environment configuration
internal/database/     SQLite connection + embedded migrations
internal/bookmark/     domain model, repository, service, HTTP handlers
internal/web/          HTML template rendering (embedded templates)
internal/server/       route registration
web/static/            Tailwind input/output CSS, served at /static
data/                  SQLite database file (gitignored)
```

## Getting started

Requires Go 1.22+ and Node.js (for the Tailwind CLI).

```sh
cp .env.example .env
npm install
make run
```

The server starts on `http://localhost:8080`. On first run it creates
`data/bookmarks.db` and applies migrations automatically.

While making CSS changes, run the Tailwind watcher in a separate terminal:

```sh
make css-watch
```

## Commands

| Command          | Description                              |
| ---------------- | ----------------------------------------- |
| `make run`       | Build CSS once and run the app            |
| `make css-watch` | Rebuild CSS on file change (dev)          |
| `make build`     | Build a production binary with minified CSS |
| `make test`      | Run the test suite                        |
| `make vet`       | Run `go vet`                              |
