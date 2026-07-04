package ws

import (
	"log"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*tokenBucket
	rate     int
	burst    int
}

type tokenBucket struct {
	tokens    float64
	lastCheck time.Time
}

func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*tokenBucket),
		rate:    rate,
		burst:   burst,
	}
}

func (rl *RateLimiter) Allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.clients[userID]
	if !ok {
		bucket = &tokenBucket{
			tokens:    float64(rl.burst),
			lastCheck: time.Now(),
		}
		rl.clients[userID] = bucket
	}

	now := time.Now()
	elapsed := now.Sub(bucket.lastCheck).Seconds()
	bucket.tokens += elapsed * float64(rl.rate)
	if bucket.tokens > float64(rl.burst) {
		bucket.tokens = float64(rl.burst)
	}
	bucket.lastCheck = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}
	log.Printf("rate limit exceeded for user=%s", userID)
	return false
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for id, bucket := range rl.clients {
		if time.Since(bucket.lastCheck) > 5*time.Minute {
			delete(rl.clients, id)
		}
	}
}
