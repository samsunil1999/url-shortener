package cache

import (
	"context"
	"fmt"
	"sync"
)

const (
	DefaultBatchSize = 1000
	CounterKey       = "url:counter"
	base62Chars      = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// RedisReserver defines the interface for reserving ID batches
type RedisReserver interface {
	ReserveBatch(ctx context.Context, key string, batchSize int64) (int64, error)
}

type Generator struct {
	mu        sync.Mutex // protects current and max
	current   int64      // next ID to hand out
	max       int64      // upper bound of current batch (exclusive)
	batchSize int64
	store     RedisReserver
}

func NewGenerator(store RedisReserver, batchSize int64) *Generator {
	return &Generator{
		batchSize: batchSize,
		store:     store,
	}
}

func (g *Generator) NextID(ctx context.Context) (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Refill if current batch is exhausted
	if g.current >= g.max {
		if err := g.refill(ctx); err != nil {
			return 0, fmt.Errorf("failed to refill ID batch: %w", err)
		}
	}

	id := g.current
	g.current++
	return id, nil
}

// refill reserves a new batch from Redis
// must be called with mu held
func (g *Generator) refill(ctx context.Context) error {
	// IncrBy returns the new max, e.g., if counter was 1000 and batchSize=1000
	// it returns 2000, meaning we own IDs 1000–1999
	newMax, err := g.store.ReserveBatch(ctx, CounterKey, g.batchSize)
	if err != nil {
		return err
	}
	g.current = newMax - g.batchSize
	g.max = newMax
	return nil
}

// ToBase62 encodes an integer ID to a base62 string
func ToBase62(num int64) string {
	if num == 0 {
		return "0"
	}
	result := []byte{}
	for num > 0 {
		result = append([]byte{base62Chars[num%62]}, result...)
		num /= 62
	}
	return string(result)
}
