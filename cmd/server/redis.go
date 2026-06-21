package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisConfig struct {
	client   *redis.Client
	cacheTTL time.Duration
}

func newRedisClient() redisConfig {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "host.docker.internal:6379"
		slog.Warn("REDIS_ADDR not set, using default local redis address")
	}

	var cacheTTL time.Duration
	if raw := os.Getenv("REDIS_CACHE_TTL"); raw == "" {
		cacheTTL = 1 * time.Hour
		slog.Warn("REDIS_CACHE_TTL not set, using default value of 1 hour")
	} else {
		var err error
		cacheTTL, err = time.ParseDuration(raw)
		if err != nil {
			cacheTTL = 1 * time.Hour
			slog.Warn("Invalid REDIS_CACHE_TTL, using default value of 1 hour")
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		slog.Error("failed to ping redis", "error", err)
		os.Exit(1)
	}

	return redisConfig{client: client, cacheTTL: cacheTTL}
}
