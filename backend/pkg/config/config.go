package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration sourced from environment variables.
// No hardcoded values — 12-Factor App compliant.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Obs      ObsConfig
	Stream   StreamConfig
	Mail     MailConfig
	S3       S3Config
	// RateLimit throttles callers by address. See RateLimitConfig.
	RateLimit RateLimitConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	// TrustedProxy tells the server it sits behind a reverse proxy that
	// terminates TLS (see docker/Caddyfile), and that X-Forwarded-For may
	// therefore be believed. It must stay false whenever the server is
	// reachable directly: any client can set that header, so trusting it on an
	// exposed socket lets a caller forge its own address in the audit log.
	TrustedProxy bool
	// CORSAllowedOrigins lists the browser origins allowed to call the API.
	// The mobile client is unaffected (CORS is a browser rule), so in
	// production this only needs the web front-end's own origin.
	CORSAllowedOrigins []string
}

// RateLimitConfig caps how often one client address may call the API.
//
// Two tiers: Requests/Window applies to the API at large, and
// AuthRequests/AuthWindow replaces it on the credential endpoints under
// /auth/, where repeated guessing is the whole attack. A tier with zero
// requests or a zero window is unlimited, so an unset variable widens the
// limit rather than taking the API offline.
//
// Meaningful only when the client address is trustworthy — see
// ServerConfig.TrustedProxy.
type RateLimitConfig struct {
	Enabled      bool
	Requests     int
	Window       time.Duration
	AuthRequests int
	AuthWindow   time.Duration
}

type DatabaseConfig struct {
	URL          string
	MaxOpenConns int
	MaxIdleConns int
}

type AuthConfig struct {
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// ObsConfig configures telemetry. Enabled=false disables every OTLP
// exporter — the process still logs to stdout, but creates no connection to a
// collector. Keep it off for local runs without the observability stack.
type ObsConfig struct {
	Enabled        bool
	LogLevel       string
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	// PrometheusURL powers the admin supervision endpoint. Empty means no
	// live metrics are served — incidents still are.
	PrometheusURL string
	// AlertWebhookSecret authenticates Grafana's alert webhook. Empty
	// disables the endpoint rather than leaving it open.
	AlertWebhookSecret string
}

type StreamConfig struct {
	MaxListeners int
	BufferSize   int
}

type MailConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

// S3Config configures track audio storage against any S3-compatible backend
// (real AWS S3 or a self-hosted server such as MinIO). Left empty, the
// server still starts — only POST /tracks/upload-url and S3 object deletion
// fail, and both sit behind the track_uploads feature flag.
//
// Endpoint is used by the backend itself (e.g. a Docker-internal MinIO
// hostname). PublicEndpoint is embedded in presigned upload URLs and in
// PublicURL()/KeyFromURL() — both are consumed directly by the mobile
// client, so it must be a host reachable from wherever the app runs
// (emulator/physical device/public internet). If PublicEndpoint is empty it
// falls back to Endpoint. If both are empty, real AWS S3 is used.
type S3Config struct {
	BucketName      string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	PublicEndpoint  string
}

// Load reads all configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
			TrustedProxy: getBool("TRUSTED_PROXY", false),
			CORSAllowedOrigins: getList("CORS_ALLOWED_ORIGINS",
				[]string{"http://localhost:*", "http://127.0.0.1:*"}),
		},
		Database: DatabaseConfig{
			URL:          requireEnv("DATABASE_URL"),
			MaxOpenConns: getInt("DATABASE_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getInt("DATABASE_MAX_IDLE_CONNS", 5),
		},
		Auth: AuthConfig{
			JWTSecret:  requireEnv("JWT_SECRET"),
			AccessTTL:  getDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: getDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Obs: ObsConfig{
			Enabled:        getBool("OTEL_ENABLED", true),
			LogLevel:       getEnv("LOG_LEVEL", "info"),
			ServiceName:    getEnv("OTEL_SERVICE_NAME", "lyo-backend"),
			ServiceVersion: getEnv("OTEL_SERVICE_VERSION", "dev"),
			Environment:    getEnv("APP_ENV", "development"),
			OTLPEndpoint:   getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"),

			PrometheusURL:      getEnv("PROMETHEUS_URL", ""),
			AlertWebhookSecret: getEnv("ALERT_WEBHOOK_SECRET", ""),
		},
		Stream: StreamConfig{
			MaxListeners: getInt("STREAM_MAX_LISTENERS", 500),
			BufferSize:   getInt("STREAM_BUFFER_SIZE", 65536),
		},
		Mail: MailConfig{
			Host: getEnv("SMTP_HOST", ""),
			Port: getInt("SMTP_PORT", 587),
			User: getEnv("SMTP_USER", ""),
			Pass: getEnv("SMTP_PASS", ""),
			From: getEnv("EMAIL_FROM", "noreply@lyo.app"),
		},
		S3: S3Config{
			BucketName:      getEnv("S3_BUCKET_NAME", ""),
			Region:          getEnv("S3_REGION", ""),
			AccessKeyID:     getEnv("S3_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
			Endpoint:        getEnv("S3_ENDPOINT", ""),
			PublicEndpoint:  getEnv("S3_PUBLIC_ENDPOINT", ""),
		},
		RateLimit: RateLimitConfig{
			Enabled:  getBool("RATE_LIMIT_ENABLED", true),
			Requests: getInt("RATE_LIMIT_REQUESTS", 120),
			Window:   getDuration("RATE_LIMIT_WINDOW", time.Minute),
			// Ten sign-in attempts a minute is far above what a person with a
			// password manager needs and far below what guessing one requires.
			AuthRequests: getInt("RATE_LIMIT_AUTH_REQUESTS", 10),
			AuthWindow:   getDuration("RATE_LIMIT_AUTH_WINDOW", time.Minute),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// minJWTSecretLen is the floor for the HMAC key. HS256 keys shorter than the
// hash output are the practical way this deployment gets forged tokens — and a
// forged token is a forged role, admin included.
const minJWTSecretLen = 32

func (c *Config) validate() error {
	if len(c.Auth.JWTSecret) < minJWTSecretLen {
		return fmt.Errorf("JWT_SECRET must be at least %d characters, got %d",
			minJWTSecretLen, len(c.Auth.JWTSecret))
	}
	if c.Obs.Environment == "production" && !c.Server.TrustedProxy {
		return fmt.Errorf("production requires TRUSTED_PROXY=true: the API must be served through the TLS reverse proxy, never exposed directly")
	}
	if c.Obs.Environment == "production" && !c.RateLimit.Enabled {
		return fmt.Errorf("production requires RATE_LIMIT_ENABLED=true: without it the credential endpoints accept unlimited guesses")
	}
	return nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

// getList reads a comma-separated variable, trimming blanks. An unset or
// all-blank value falls back rather than yielding an empty allow-list, which
// for CORS would silently mean "allow nothing".
func getList(key string, fallback []string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	out := make([]string, 0, strings.Count(raw, ",")+1)
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
