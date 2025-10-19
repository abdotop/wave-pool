package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/abdotop/wave-pool/db/sqlc"
	"github.com/abdotop/wave-pool/domain"
	"github.com/redis/go-redis/v9"
)

// RedisClient defines the interface for Redis operations.
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type API struct {
	db            sqlc.Querier
	redis         RedisClient
	webhookSender *WebhookSender
}

func NewAPI(db sqlc.Querier, redis RedisClient) *API {
	return &API{
		db:            db,
		redis:         redis,
		webhookSender: NewWebhookSender(db.(*sqlc.Queries)),
	}
}

func returnError(w http.ResponseWriter, err domain.LastPaymentError, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(err)
}
