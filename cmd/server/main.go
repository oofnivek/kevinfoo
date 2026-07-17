// Command server runs the bookmarks web application.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bookmarks/internal/bookmark"
	"bookmarks/internal/config"
	"bookmarks/internal/database"
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

	repo, ping, closeDB, err := openRepository(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer closeDB()

	renderer, err := web.NewRenderer()
	if err != nil {
		return err
	}

	svc := bookmark.NewService(repo)
	handler := bookmark.NewHandler(svc, renderer, logger)

	mux := server.NewMux(handler, renderer, ping, "web/static", logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

// openRepository connects to the configured storage backend and returns the
// bookmark repository, a health-check ping function, and a cleanup func.
func openRepository(ctx context.Context, cfg config.Config) (bookmark.Repository, func(context.Context) error, func(), error) {
	switch cfg.DBDriver {
	case "mongo":
		client, err := database.OpenMongo(cfg.MongoURI())
		if err != nil {
			return nil, nil, nil, err
		}

		repo, err := bookmark.NewMongoRepository(ctx, client.Database(cfg.MongoDatabase))
		if err != nil {
			client.Disconnect(context.Background())
			return nil, nil, nil, err
		}

		ping := func(ctx context.Context) error { return client.Ping(ctx, nil) }
		closeFn := func() { client.Disconnect(context.Background()) }
		return repo, ping, closeFn, nil

	case "sqlite":
		db, err := database.Open(cfg.DBPath)
		if err != nil {
			return nil, nil, nil, err
		}

		if err := database.Migrate(db); err != nil {
			db.Close()
			return nil, nil, nil, err
		}

		repo := bookmark.NewSQLiteRepository(db)
		return repo, db.PingContext, func() { db.Close() }, nil

	default:
		return nil, nil, nil, fmt.Errorf("unknown DB_DRIVER %q (expected \"sqlite\" or \"mongo\")", cfg.DBDriver)
	}
}
