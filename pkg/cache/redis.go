package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	Client *redis.Client
}

func NewRedisStore(addr string) (*RedisStore, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &RedisStore{Client: rdb}, nil
}

func (r *RedisStore) ReserveBatch(ctx context.Context, key string, batchSize int64) (int64, error) {
	return r.Client.IncrBy(ctx, key, batchSize).Result()
}

func (r *RedisStore) Close() error {
	return r.Client.Close()
}
