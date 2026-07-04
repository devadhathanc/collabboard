package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Service struct {
	rdb      *redis.Client
	mu       sync.Mutex
	inflight map[string]chan struct{}
}

func NewService(rdb *redis.Client) *Service {
	return &Service{
		rdb:      rdb,
		inflight: make(map[string]chan struct{}),
	}
}

func (s *Service) GetOrFetch(ctx context.Context, key string, ttl time.Duration, fetch func() (interface{}, error), dest interface{}) error {
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err == nil {
		return json.Unmarshal(data, dest)
	}

	s.mu.Lock()
	if ch, ok := s.inflight[key]; ok {
		s.mu.Unlock()
		<-ch
		data, err := s.rdb.Get(ctx, key).Bytes()
		if err == nil {
			return json.Unmarshal(data, dest)
		}
		return s.fetchAndStore(ctx, key, ttl, fetch, dest)
	}

	ch := make(chan struct{})
	s.inflight[key] = ch
	s.mu.Unlock()

	err = s.fetchAndStore(ctx, key, ttl, fetch, dest)

	s.mu.Lock()
	delete(s.inflight, key)
	close(ch)
	s.mu.Unlock()

	return err
}

func (s *Service) fetchAndStore(ctx context.Context, key string, ttl time.Duration, fetch func() (interface{}, error), dest interface{}) error {
	result, err := fetch()
	if err != nil {
		return err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	if err := s.rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Printf("cache set error: %v", err)
	}

	return json.Unmarshal(data, dest)
}

func (s *Service) Invalidate(ctx context.Context, keys ...string) {
	for _, key := range keys {
		if err := s.rdb.Del(ctx, key).Err(); err != nil {
			log.Printf("cache invalidation error for key=%s: %v", key, err)
		}
	}
}

func (s *Service) PublishInvalidation(ctx context.Context, channel string, keys ...string) {
	msg, _ := json.Marshal(keys)
	if err := s.rdb.Publish(ctx, channel, msg).Err(); err != nil {
		log.Printf("cache invalidation publish error: %v", err)
	}
}

func TaskCacheKey(boardID, taskID string) string {
	return fmt.Sprintf("cache:task:%s:%s", boardID, taskID)
}

func BoardTasksCacheKey(boardID string) string {
	return fmt.Sprintf("cache:board:%s:tasks", boardID)
}
