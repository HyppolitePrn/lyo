package config

import (
	"fmt"
	"os"
	"strconv"
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
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
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
	}
	return cfg, nil
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

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
