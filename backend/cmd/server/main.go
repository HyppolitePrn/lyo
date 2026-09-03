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
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/hyppoliteprn/lyo/internal/api"
	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/features"
	"github.com/hyppoliteprn/lyo/internal/incident"
	"github.com/hyppoliteprn/lyo/internal/observability"
	"github.com/hyppoliteprn/lyo/internal/passwordreset"
	"github.com/hyppoliteprn/lyo/internal/playlist"
	"github.com/hyppoliteprn/lyo/internal/storage"
	"github.com/hyppoliteprn/lyo/internal/streaming"
	"github.com/hyppoliteprn/lyo/internal/track"
	"github.com/hyppoliteprn/lyo/internal/user"
	userusecase "github.com/hyppoliteprn/lyo/internal/user/usecase"
	"github.com/hyppoliteprn/lyo/migrations"
	"github.com/hyppoliteprn/lyo/pkg/config"
	"github.com/hyppoliteprn/lyo/pkg/mailer"
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

	// Telemetry first: everything below logs through the provider it returns,
	// so traces, metrics and logs share one resource identity from line one.
	otelCtx, otelCancel := context.WithTimeout(context.Background(), 15*time.Second)
	obs, err := observability.Setup(otelCtx, observability.Config{
		Enabled:        cfg.Obs.Enabled,
		ServiceName:    cfg.Obs.ServiceName,
		ServiceVersion: cfg.Obs.ServiceVersion,
		Environment:    cfg.Obs.Environment,
		OTLPEndpoint:   cfg.Obs.OTLPEndpoint,
		LogLevel:       cfg.Obs.LogLevel,
	})
	otelCancel()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "observability error: %v\n", err)
		os.Exit(1)
	}
	logger := obs.Logger

	metrics, err := observability.NewMetrics()
	if err != nil {
		logger.Error("cannot register metrics", "err", err)
		os.Exit(1)
	}

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
	stdDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = stdDB.Close() }()

	if err := runMigrations(stdDB); err != nil {
		logger.Error("migration failed", "err", err)
		os.Exit(1)
	}
	logger.Info("migrations applied")

	// ── Auth & router ─────────────────────────────────────────────────────────
	authSvc := auth.NewService(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL, cfg.Auth.RefreshTTL)

	userRepo := user.NewRepository(pool)
	userSvc := user.NewService(userRepo, authSvc)
	getUserByIDUC := userusecase.NewGetUserByIDUsecase(userRepo)
	updateUserByIDUC := userusecase.NewUpdateUserByIDUsecase(userRepo)

	featRepo := features.NewRepository(pool)
	featSvc := features.NewService(featRepo)

	mailSvc := mailer.New(cfg.Mail)
	pwResetRepo := passwordreset.NewRepository(pool)
	pwResetSvc := passwordreset.NewService(pwResetRepo, userRepo, mailSvc, logger)

	streamRepo := streaming.NewRepository(pool)
	streamSvc := streaming.NewService(streamRepo, cfg.Stream.BufferSize, logger, metrics)

	s3Storage, err := storage.New(cfg.S3)
	if err != nil {
		logger.Error("cannot init s3 storage", "err", err)
		os.Exit(1)
	}
	trackRepo := track.NewRepository(pool)
	trackSvc := track.NewService(trackRepo, s3Storage)
	playlistRepo := playlist.NewRepository(pool)
	playlistSvc := playlist.NewService(playlistRepo)

	incidentRepo := incident.NewRepository(pool)
	incidentSvc := incident.NewService(incidentRepo, mailSvc, logger)
	// nil when PROMETHEUS_URL is unset: the supervision endpoint then serves
	// incidents without live metrics rather than failing.
	promClient := observability.NewPrometheusClient(cfg.Obs.PrometheusURL)

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
	r.Use(middleware.Trace(cfg.Obs.ServiceName))
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Authenticate(authSvc))

	// Mount generated API routes
	strict := api.NewStrictHandlerWithOptions(
		api.NewHandlers(userSvc, authSvc, streamSvc, featSvc, pwResetSvc, trackSvc, playlistSvc,
			getUserByIDUC, updateUserByIDUC, incidentSvc, metricsQuerier(promClient), logger),
		nil,
		api.StrictHTTPServerOptions{
			ResponseErrorHandlerFunc: handleResponseError,
			RequestErrorHandlerFunc:  handleRequestError,
		},
	)
	api.HandlerFromMux(strict, r)

	// WebSocket endpoints (out-of-band, not in OpenAPI spec)
	ingestH := streaming.NewIngestHandler(streamSvc, authSvc, logger, metrics)
	r.Get("/streams/{id}/ingest", ingestH.ServeHTTP)

	listenH := streaming.NewListenHandler(streamSvc, authSvc, logger, metrics)
	r.Get("/streams/{id}/listen", listenH.ServeHTTP)

	// Grafana's alert webhook. Out of the OpenAPI contract for the same reason
	// as the WebSocket endpoints: the body is Grafana's schema, and the caller
	// is infrastructure that cannot mint a JWT.
	alertH := incident.NewWebhookHandler(incidentSvc, cfg.Obs.AlertWebhookSecret, logger)
	r.Post("/internal/alerts", alertH.ServeHTTP)

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
	// Flush buffered spans, metrics and logs — otherwise the last window of
	// telemetry before a deploy, which is exactly the interesting one, is lost.
	if err := obs.Shutdown(ctx); err != nil {
		logger.Error("telemetry shutdown error", "err", err)
	}
}

// metricsQuerier converts a possibly-nil client into a possibly-nil interface.
// Assigning a nil *PrometheusClient straight to the interface would produce a
// non-nil interface holding a nil pointer, and the handler's nil check would
// no longer fire.
func metricsQuerier(c *observability.PrometheusClient) api.MetricsQuerier {
	if c == nil {
		return nil
	}
	return c
}

func runMigrations(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("migrations source: %w", err)
	}
	driver, err := migratepostgres.WithInstance(db, &migratepostgres.Config{})
	if err != nil {
		_ = src.Close()
		return fmt.Errorf("migrate driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		_ = src.Close()
		return fmt.Errorf("migrate init: %w", err)
	}
	// m.Close() closes both src and driver.
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
