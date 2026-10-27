// Command server runs the bookmarks web application.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bookmarks/internal/auth"
	"bookmarks/internal/bookmark"
	"bookmarks/internal/config"
	"bookmarks/internal/database"
	"bookmarks/internal/loginlog"
	"bookmarks/internal/push"
	"bookmarks/internal/ratelimit"
	"bookmarks/internal/server"
	"bookmarks/internal/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()

	if cfg.AuthUsername == "" || cfg.AuthPassword == "" {
		return fmt.Errorf("AUTH_USERNAME and AUTH_PASSWORD must be set")
	}

	repos, ping, closeDB, err := openRepositories(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer closeDB()

	attempts, closeAttempts, err := openLoginLogger(context.Background(), cfg, logger)
	if err != nil {
		return err
	}
	defer closeAttempts()

	renderer, err := web.NewRenderer()
	if err != nil {
		return err
	}

	svc := bookmark.NewService(repos.bookmarks)
	handler := bookmark.NewHandler(svc, renderer, logger)

	sessionSecret := cfg.SessionSecret
	if sessionSecret == "" {
		logger.Warn("SESSION_SECRET not set, generating an ephemeral one; sessions will not survive a restart")
		sessionSecret = randomSecret()
	}

	recaptchaSiteKey, recaptchaSecretKey := cfg.RecaptchaSiteKey, cfg.RecaptchaSecretKey
	if !cfg.RecaptchaEnabled {
		recaptchaSiteKey, recaptchaSecretKey = "", ""
	}

	session := auth.New(sessionSecret)
	authHandler := auth.NewHandler(session, cfg.AuthUsername, cfg.AuthPassword, recaptchaSiteKey, recaptchaSecretKey, attempts, renderer, logger)

	if cfg.VAPIDPublicKey == "" || cfg.VAPIDPrivateKey == "" {
		logger.Warn("VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY not set, push notifications are disabled; generate keys with: make vapid")
	} else if cfg.VAPIDSubject == "" {
		logger.Warn("VAPID_SUBJECT not set; some push services (e.g. Safari) reject notifications without a contact address")
	}
	pushSvc := push.NewService(repos.push, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject, logger)
	pushHandler := push.NewHandler(pushSvc, renderer, logger)

	// Public push endpoints are limited to one request per second per IP.
	pushLimiter := ratelimit.New(time.Second)

	mux := server.NewMux(handler, pushHandler, authHandler, session.Middleware, pushLimiter.Middleware, session.Valid, renderer, ping, "web/static", logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go pushSvc.RunCleanup(ctx, time.Hour)

	go func() {
		<-ctx.Done()
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	}()

	logger.Info("starting server", "port", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func randomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// openLoginLogger connects to MongoDB and returns a logger for login
// attempts. This is independent of DBDriver: login attempts are audited to
// MongoDB even when bookmarks are stored in SQLite. If no Mongo host is
// configured, login attempts are silently not logged, so local/sqlite-only
// setups keep working without Mongo credentials.
func openLoginLogger(ctx context.Context, cfg config.Config, logger *slog.Logger) (loginlog.Logger, func(), error) {
	if cfg.MongoHost == "" {
		logger.Warn("MONGODB_HOST not set, login attempts will not be logged")
		return nil, func() {}, nil
	}

	client, err := database.OpenMongo(cfg.MongoURI())
	if err != nil {
		return nil, nil, err
	}

	attemptLogger, err := loginlog.NewMongoLogger(ctx, client.Database(cfg.MongoDatabase))
	if err != nil {
		client.Disconnect(context.Background())
		return nil, nil, err
	}

	return attemptLogger, func() { client.Disconnect(context.Background()) }, nil
}

// repositories holds the storage for each feature, all backed by the same
// configured database.
type repositories struct {
	bookmarks bookmark.Repository
	push      push.Repository
}

// openRepositories connects to the configured storage backend and returns the
// repositories, a health-check ping function, and a cleanup func.
func openRepositories(ctx context.Context, cfg config.Config) (repositories, func(context.Context) error, func(), error) {
	switch cfg.DBDriver {
	case "mongo":
		client, err := database.OpenMongo(cfg.MongoURI())
		if err != nil {
			return repositories{}, nil, nil, err
		}
		bookmarks, err := bookmark.NewMongoRepository(ctx, client.Database(cfg.MongoDatabase))
		if err != nil {
			client.Disconnect(context.Background())
			return repositories{}, nil, nil, err
		}

		pushes, err := push.NewMongoRepository(ctx, client.Database(cfg.MongoPushDatabase))
		if err != nil {
			client.Disconnect(context.Background())
			return repositories{}, nil, nil, err
		}

		ping := func(ctx context.Context) error { return client.Ping(ctx, nil) }
		closeFn := func() { client.Disconnect(context.Background()) }
		return repositories{bookmarks: bookmarks, push: pushes}, ping, closeFn, nil

	case "sqlite":
		db, err := database.Open(cfg.DBPath)
		if err != nil {
			return repositories{}, nil, nil, err
		}

		if err := database.Migrate(db); err != nil {
			db.Close()
			return repositories{}, nil, nil, err
		}

		repos := repositories{
			bookmarks: bookmark.NewSQLiteRepository(db),
			push:      push.NewSQLiteRepository(db),
		}
		return repos, db.PingContext, func() { db.Close() }, nil

	default:
		return repositories{}, nil, nil, fmt.Errorf("unknown DB_DRIVER %q (expected \"sqlite\" or \"mongo\")", cfg.DBDriver)
	}
}
