package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb      *redis.Client
	perIP    int
	perAcct  int
	window   time.Duration
}

func NewRateLimiter(rdb *redis.Client, perIP, perAcct int) *RateLimiter {
	return &RateLimiter{
		rdb:     rdb,
		perIP:   perIP,
		perAcct: perAcct,
		window:  60 * time.Second,
	}
}

func (rl *RateLimiter) Middleware(strict bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)
			key := "ratelimit:ip:" + ip

			allowed, err := rl.allow(r.Context(), key, rl.perIP)
			if err != nil {
				if strict {
					http.Error(w, `{"error":"rate limited"}`, http.StatusTooManyRequests)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				http.Error(w, `{"error":"rate limited"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) allow(ctx context.Context, key string, limit int) (bool, error) {
	pipe := rl.rdb.Pipeline()
	now := time.Now().Unix()
	window := int64(rl.window.Seconds())
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", now-window))
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
	pipe.Expire(ctx, key, rl.window)
	count, err := pipe.ZCard(ctx, key).Result()
	if err != nil {
		return false, err
	}
	pipe.Exec(ctx)
	return count <= int64(limit), nil
}

func extractIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	idx := strings.LastIndex(r.RemoteAddr, ":")
	if idx > 0 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}
