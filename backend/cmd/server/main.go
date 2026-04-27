package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	migratesource "github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/hyppoliteprn/lyo/internal/api"
	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/features"
	"github.com/hyppoliteprn/lyo/internal/observability"
	"github.com/hyppoliteprn/lyo/internal/streaming"
	"github.com/hyppoliteprn/lyo/internal/user"
	"github.com/hyppoliteprn/lyo/migrations"
	"github.com/hyppoliteprn/lyo/pkg/config"
	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSONError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if encodeErr := json.NewEncoder(w).Encode(errorResponse{Error: err.Error()}); encodeErr != nil {
		http.Error(w, http.StatusText(status), status)
	}
}

func handleResponseError(w http.ResponseWriter, _ *http.Request, err error) {
	if he, ok := errors.AsType[*api.HTTPError](err); ok {
		writeJSONError(w, he.Code, errors.New(he.Msg))
		return
	}
	writeJSONError(w, http.StatusNotImplemented, err)
}

func handleRequestError(w http.ResponseWriter, _ *http.Request, err error) {
	writeJSONError(w, http.StatusBadRequest, err)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.Obs.LogLevel)

	// ── Database pool ────────────────────────────────────────────────────────
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		logger.Error("invalid DATABASE_URL", "err", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = int32(cfg.Database.MaxOpenConns) //nolint:gosec
	poolCfg.MinConns = int32(cfg.Database.MaxIdleConns) //nolint:gosec

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		logger.Error("cannot open db pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// ── Auto-migrate ─────────────────────────────────────────────────────────
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		logger.Error("migrations source error", "err", err)
		os.Exit(1)
	}

	stdDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = stdDB.Close() }()

	if err := runMigrations(stdDB, src); err != nil {
		logger.Error("migration failed", "err", err)
		os.Exit(1)
	}
	logger.Info("migrations applied")

	// ── Auth & router ─────────────────────────────────────────────────────────
	authSvc := auth.NewService(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL, cfg.Auth.RefreshTTL)

	userRepo := user.NewRepository(pool)
	userSvc := user.NewService(userRepo, authSvc)

	featRepo := features.NewRepository(pool)
	featSvc := features.NewService(featRepo)

	streamRepo := streaming.NewRepository(pool)
	streamSvc := streaming.NewService(streamRepo, cfg.Stream.BufferSize, logger)

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Authenticate(authSvc))

	// Mount generated API routes
	strict := api.NewStrictHandlerWithOptions(
		api.NewHandlers(userSvc, authSvc, streamSvc, featSvc, logger),
		nil,
		api.StrictHTTPServerOptions{
			ResponseErrorHandlerFunc: handleResponseError,
			RequestErrorHandlerFunc:  handleRequestError,
		},
	)
	api.HandlerFromMux(strict, r)

	// WebSocket audio ingest (out-of-band, not in OpenAPI spec)
	ingestH := streaming.NewIngestHandler(streamSvc, authSvc, logger)
	r.Get("/streams/{id}/ingest", ingestH.ServeHTTP)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
}

func runMigrations(db *sql.DB, src migratesource.Driver) error {
	driver, err := migratepostgres.WithInstance(db, &migratepostgres.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		return fmt.Errorf("migrate close source: %w", srcErr)
	}
	if dbErr != nil {
		return fmt.Errorf("migrate close db: %w", dbErr)
	}
	return nil
}
