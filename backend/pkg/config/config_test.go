package config_test

import (
	"testing"
	"time"

	"github.com/hyppoliteprn/lyo/pkg/config"
)

// optionalVars is every variable Load reads that has a default. They are
// cleared before each test so a developer's own environment (e.g. a direnv
// .envrc) cannot change what the defaults test observes.
var optionalVars = []string{
	"SERVER_PORT", "SERVER_READ_TIMEOUT", "SERVER_WRITE_TIMEOUT", "SERVER_IDLE_TIMEOUT",
	"DATABASE_MAX_OPEN_CONNS", "DATABASE_MAX_IDLE_CONNS",
	"JWT_ACCESS_TTL", "JWT_REFRESH_TTL",
	"LOG_LEVEL", "OTEL_SERVICE_NAME", "OTEL_EXPORTER_OTLP_ENDPOINT",
	"STREAM_MAX_LISTENERS", "STREAM_BUFFER_SIZE",
	"SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASS", "EMAIL_FROM",
	"S3_BUCKET_NAME", "S3_REGION", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY",
	"S3_ENDPOINT", "S3_PUBLIC_ENDPOINT",
}

// setRequired sets the two variables Load panics without and clears the rest.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/lyo")
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-chars!")
	for _, v := range optionalVars {
		t.Setenv(v, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	setRequired(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if cfg.Server.Port != "8080" {
		t.Errorf("port = %q, want 8080", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("read timeout = %v, want 30s", cfg.Server.ReadTimeout)
	}
	if cfg.Server.IdleTimeout != 120*time.Second {
		t.Errorf("idle timeout = %v, want 120s", cfg.Server.IdleTimeout)
	}
	if cfg.Database.MaxOpenConns != 25 || cfg.Database.MaxIdleConns != 5 {
		t.Errorf("db conns = %d/%d, want 25/5", cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns)
	}
	if cfg.Auth.AccessTTL != 15*time.Minute || cfg.Auth.RefreshTTL != 7*24*time.Hour {
		t.Errorf("ttls = %v/%v", cfg.Auth.AccessTTL, cfg.Auth.RefreshTTL)
	}
	if cfg.Obs.LogLevel != "info" || cfg.Obs.ServiceName != "lyo-backend" {
		t.Errorf("obs = %+v", cfg.Obs)
	}
	if cfg.Stream.MaxListeners != 500 || cfg.Stream.BufferSize != 65536 {
		t.Errorf("stream = %+v", cfg.Stream)
	}
	if cfg.Mail.Port != 587 || cfg.Mail.From != "noreply@lyo.app" {
		t.Errorf("mail = %+v", cfg.Mail)
	}
	if cfg.S3.BucketName != "" || cfg.S3.Endpoint != "" {
		t.Errorf("s3 should default to empty, got %+v", cfg.S3)
	}
}

func TestLoad_OverridesFromEnv(t *testing.T) {
	setRequired(t)
	t.Setenv("SERVER_PORT", "9000")
	t.Setenv("SERVER_READ_TIMEOUT", "5s")
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "50")
	t.Setenv("JWT_ACCESS_TTL", "1h")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("STREAM_BUFFER_SIZE", "1024")
	t.Setenv("SMTP_HOST", "mail.example.com")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("S3_BUCKET_NAME", "lyo-tracks")
	t.Setenv("S3_PUBLIC_ENDPOINT", "http://minio.local:9000")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if cfg.Server.Port != "9000" || cfg.Server.ReadTimeout != 5*time.Second {
		t.Errorf("server = %+v", cfg.Server)
	}
	if cfg.Database.MaxOpenConns != 50 {
		t.Errorf("max open conns = %d, want 50", cfg.Database.MaxOpenConns)
	}
	if cfg.Auth.AccessTTL != time.Hour {
		t.Errorf("access ttl = %v, want 1h", cfg.Auth.AccessTTL)
	}
	if cfg.Obs.LogLevel != "debug" {
		t.Errorf("log level = %q", cfg.Obs.LogLevel)
	}
	if cfg.Stream.BufferSize != 1024 {
		t.Errorf("buffer size = %d, want 1024", cfg.Stream.BufferSize)
	}
	if cfg.Mail.Host != "mail.example.com" || cfg.Mail.Port != 2525 {
		t.Errorf("mail = %+v", cfg.Mail)
	}
	if cfg.S3.BucketName != "lyo-tracks" || cfg.S3.PublicEndpoint != "http://minio.local:9000" {
		t.Errorf("s3 = %+v", cfg.S3)
	}
}

// Unparseable numeric/duration values fall back to the default rather than failing.
func TestLoad_MalformedValuesFallBackToDefaults(t *testing.T) {
	setRequired(t)
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "not-a-number")
	t.Setenv("SERVER_WRITE_TIMEOUT", "not-a-duration")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("max open conns = %d, want default 25", cfg.Database.MaxOpenConns)
	}
	if cfg.Server.WriteTimeout != 30*time.Second {
		t.Errorf("write timeout = %v, want default 30s", cfg.Server.WriteTimeout)
	}
}

func TestLoad_PanicsWhenRequiredVarMissing(t *testing.T) {
	for _, missing := range []string{"DATABASE_URL", "JWT_SECRET"} {
		t.Run(missing, func(t *testing.T) {
			setRequired(t)
			t.Setenv(missing, "")

			defer func() {
				if recover() == nil {
					t.Fatalf("expected Load to panic when %s is unset", missing)
				}
			}()
			_, _ = config.Load()
		})
	}
}
