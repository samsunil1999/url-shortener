package cache

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// SyncCounterFromDB seeds the Redis counter from the DB max ID on startup.
// This ensures counter continuity if Redis loses its data.
func SyncCounterFromDB(ctx context.Context, rdb *redis.Client, db *sql.DB) error {
	var maxID int64
	err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM urls").Scan(&maxID)
	if err != nil {
		return fmt.Errorf("failed to query max ID from DB: %w", err)
	}

	// Only update Redis if DB has a higher value (avoid going backwards)
	script := redis.NewScript(`
        local current = tonumber(redis.call("GET", KEYS[1])) or 0
        local dbMax = tonumber(ARGV[1])
        if dbMax > current then
            redis.call("SET", KEYS[1], dbMax)
        end
        return redis.call("GET", KEYS[1])
    `)

	if err := script.Run(ctx, rdb, []string{CounterKey}, maxID).Err(); err != nil {
		return fmt.Errorf("failed to sync counter to Redis: %w", err)
	}

	fmt.Printf("Redis counter synced to: %d\n", maxID)
	return nil
}
