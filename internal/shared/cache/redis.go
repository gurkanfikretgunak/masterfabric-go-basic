package cache

import (
	"context"
	"fmt"

	"github.com/masterfabric/masterfabric_go_basic/internal/shared/config"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates and pings a Redis client.
// If cfg.URL is set (e.g. a Render-provided redis://... connection string) it
// takes precedence over cfg.Addr, cfg.Password, and cfg.DB.
func NewRedisClient(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	var opts *redis.Options

	if cfg.URL != "" {
		parsed, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("parse redis url: %w", err)
		}
		opts = parsed
	} else {
		opts = &redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		}
	}

	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
