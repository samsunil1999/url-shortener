package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RateLimit(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		ip := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s", ip)
		now := time.Now().UnixMilli()
		windowStart := now - 60000 // 1-minute sliding window

		pipe := rdb.Pipeline()
		pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart, 10))
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
		pipe.ZCard(ctx, key)
		pipe.Expire(ctx, key, 2*time.Minute)

		results, err := pipe.Exec(ctx)
		if err != nil {
			c.Next() // fail open
			return
		}

		count := results[2].(*redis.IntCmd).Val()
		if count > 100 {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded, try again in a minute"})
			c.Abort()
			return
		}
		c.Next()
	}
}
