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

type ObsConfig struct {
	LogLevel        string
	ServiceName     string
	OTLPEndpoint    string
}

type StreamConfig struct {
	MaxListeners int
	BufferSize   int
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
			LogLevel:     getEnv("LOG_LEVEL", "info"),
			ServiceName:  getEnv("OTEL_SERVICE_NAME", "lyo-backend"),
			OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"),
		},
		Stream: StreamConfig{
			MaxListeners: getInt("STREAM_MAX_LISTENERS", 500),
			BufferSize:   getInt("STREAM_BUFFER_SIZE", 65536),
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

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
