package handlers

import (
	"context"
	"time"

	"github.com/abdotop/wave-pool/db/sqlc"
	"github.com/redis/go-redis/v9"
)

// RedisClient defines the interface for Redis operations.
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type API struct {
	db    sqlc.Querier
	redis RedisClient
}

func NewAPI(db sqlc.Querier, redis RedisClient) *API {
	return &API{db: db, redis: redis}
}
