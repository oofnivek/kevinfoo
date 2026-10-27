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
internal/push/         web push subscriptions, sending, service worker
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
| `make vapid`     | Generate VAPID keys for push notifications |

## Push notifications API

Subscribe a browser to a user ID + topic on `/push-notifications`, then send
to it from anywhere with `POST /push/send`. No login is needed: knowing the
user ID and topic is what authorizes a send, so keep the topic secret.

| Field     | Required | Description                                   |
| --------- | -------- | --------------------------------------------- |
| `userId`  | yes      | UUID                                          |
| `topic`   | yes      | UUID, or 20-64 letters, digits, `-` or `_`    |
| `title`   | yes      | Up to 200 characters                          |
| `message` | no       | Up to 1000 characters                         |

Send as form fields or JSON:

```sh
curl https://kevinfoo.onrender.com/push/send \
  -d userId=<user-id> -d topic=<topic> \
  -d title=some_title -d message=some_message

curl https://kevinfoo.onrender.com/push/send \
  -H 'Content-Type: application/json' \
  -d '{"userId":"<user-id>","topic":"<topic>","title":"some_title","message":"some_message"}'
```

Responses:

| Status | Meaning                                                                  |
| ------ | ------------------------------------------------------------------------ |
| 200    | `{"sent":1,"failed":0,"removed":0}`; `removed` counts expired browsers   |
| 400    | Invalid user ID, topic, title, or message                                |
| 404    | No browser is subscribed under that user ID + topic                      |
| 429    | Rate limited: one request per second per IP                              |
| 503    | Push isn't configured on the server (VAPID keys missing)                 |
