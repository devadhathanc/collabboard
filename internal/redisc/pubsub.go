package redisc

import (
	"context"
	"fmt"

	"github.com/devadhathan/collabboard/internal/ws"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(ctx context.Context, url string) (*Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) Publish(ctx context.Context, channel string, msg []byte) error {
	return c.rdb.Publish(ctx, channel, msg).Err()
}

func (c *Client) Subscribe(ctx context.Context, channel string) ws.Subscription {
	return &pubSubAdapter{ps: c.rdb.Subscribe(ctx, channel)}
}

func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.rdb.LRange(ctx, key, start, stop).Result()
}

func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.rdb.LPush(ctx, key, values...).Err()
}

func (c *Client) LTrim(ctx context.Context, key string, start, stop int64) error {
	return c.rdb.LTrim(ctx, key, start, stop).Err()
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.rdb.Exists(ctx, keys...).Result()
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration interface{}) error {
	return c.rdb.Set(ctx, key, value, 0).Err()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

func (c *Client) Raw() *redis.Client {
	return c.rdb
}

func BoardChannel(boardID string) string {
	return "board:" + boardID
}

type pubSubAdapter struct {
	ps *redis.PubSub
}

func (a *pubSubAdapter) Channel() <-chan ws.Message {
	ch := a.ps.Channel()
	out := make(chan ws.Message, 256)
	go func() {
		for msg := range ch {
			out <- ws.Message{
				Channel: msg.Channel,
				Payload: msg.Payload,
			}
		}
		close(out)
	}()
	return out
}

func (a *pubSubAdapter) Close() error {
	return a.ps.Close()
}
