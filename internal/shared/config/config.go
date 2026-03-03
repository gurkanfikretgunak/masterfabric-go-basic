package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Cache    CacheConfig
	RabbitMQ RabbitMQConfig
	JWT      JWTConfig
	Log      LogConfig
	GraphQL  GraphQLConfig
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	DSN         string
	MaxConns    int32
	MinConns    int32
	MaxConnIdle time.Duration
}

type RedisConfig struct {
	// URL is a full redis://... connection string (e.g. from Render).
	// When set it takes precedence over Addr, Password, and DB.
	URL      string
	Addr     string
	Password string
	DB       int
}

// CacheConfig controls cache behaviour independent of the Redis connection.
// TTL fields govern how long each class of entry lives in the cache.
// KeyPrefix is prepended to every key (useful when multiple apps share a
// Redis instance — leave empty to use the built-in "mf:" namespace).
type CacheConfig struct {
	// KeyPrefix is an optional additional namespace prefix (default: "").
	KeyPrefix string

	// DefaultTTL is used when no more-specific TTL applies.
	DefaultTTL time.Duration

	// TokenBlacklistTTL caps how long a blacklisted access token is tracked.
	// Should be at least as long as JWT.AccessTokenTTL.
	TokenBlacklistTTL time.Duration

	// RefreshTokenTTL is the lifetime of refresh tokens in the cache.
	// Should match JWT.RefreshTokenTTL.
	RefreshTokenTTL time.Duration

	// UserSettingsTTL is how long per-user settings are cached.
	UserSettingsTTL time.Duration

	// AppSettingsTTL is how long application-wide settings are cached.
	AppSettingsTTL time.Duration
}

type RabbitMQConfig struct {
	URL      string
	Enabled  bool
	Exchange string
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type LogConfig struct {
	Level  string
	Format string
}

type GraphQLConfig struct {
	// Introspection controls whether the GraphQL introspection endpoint is
	// enabled. Disable in production to reduce the attack surface.
	Introspection bool
}

// Load reads configuration from environment variables with sane defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			DSN:         getEnv("DATABASE_DSN", "postgres://masterfabric:masterfabric@localhost:5432/masterfabric_basic?sslmode=disable"),
			MaxConns:    int32(getEnvInt("DATABASE_MAX_CONNS", 20)),
			MinConns:    int32(getEnvInt("DATABASE_MIN_CONNS", 2)),
			MaxConnIdle: getEnvDuration("DATABASE_MAX_CONN_IDLE", 5*time.Minute),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", ""),
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Cache: CacheConfig{
			KeyPrefix:         getEnv("CACHE_KEY_PREFIX", ""),
			DefaultTTL:        getEnvDuration("CACHE_DEFAULT_TTL", 5*time.Minute),
			TokenBlacklistTTL: getEnvDuration("CACHE_TOKEN_BLACKLIST_TTL", 15*time.Minute),
			RefreshTokenTTL:   getEnvDuration("CACHE_REFRESH_TOKEN_TTL", 7*24*time.Hour),
			UserSettingsTTL:   getEnvDuration("CACHE_USER_SETTINGS_TTL", 10*time.Minute),
			AppSettingsTTL:    getEnvDuration("CACHE_APP_SETTINGS_TTL", 30*time.Minute),
		},
		RabbitMQ: RabbitMQConfig{
			URL:      getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			Enabled:  getEnvBool("RABBITMQ_ENABLED", true),
			Exchange: getEnv("RABBITMQ_EXCHANGE", "masterfabric.events"),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "change-me-in-production-at-least-32-chars"),
			AccessTokenTTL:  getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTokenTTL: getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		GraphQL: GraphQLConfig{
			Introspection: getEnvBool("GRAPHQL_INTROSPECTION", true),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
