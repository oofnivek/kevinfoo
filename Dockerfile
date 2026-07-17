# --- CSS build stage -------------------------------------------------------
FROM node:22-alpine AS css-builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY web/static/css/input.css web/static/css/input.css
COPY internal/web/templates internal/web/templates
RUN npm run css:prod

# --- Go build stage ----------------------------------------------------------
FROM golang:1.26-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/bin/server ./cmd/server

# --- Runtime stage -----------------------------------------------------------
FROM alpine:3.22
RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=go-builder /app/bin/server ./server
COPY --from=css-builder /app/web/static/css/output.css web/static/css/output.css
RUN mkdir -p /data && chown -R app:app /app /data
USER app

ENV PORT=8080 \
    DB_PATH=/data/bookmarks.db
EXPOSE 8080

ENTRYPOINT ["./server"]
