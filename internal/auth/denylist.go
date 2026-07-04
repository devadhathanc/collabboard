package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisDenylist struct {
	rdb *redis.Client
}

func NewRedisDenylist(rdb *redis.Client) *RedisDenylist {
	return &RedisDenylist{rdb: rdb}
}

func (d *RedisDenylist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	return d.rdb.Set(ctx, "denylist:"+jti, "1", ttl).Err()
}

func (d *RedisDenylist) IsDenied(ctx context.Context, jti string) (bool, error) {
	n, err := d.rdb.Exists(ctx, "denylist:"+jti).Result()
	if err != nil {
		return false, fmt.Errorf("check denylist: %w", err)
	}
	return n > 0, nil
}
