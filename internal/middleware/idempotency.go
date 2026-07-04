package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"
)

type ctxKey string

const idempotencyKey ctxKey = "idempotency_key"

func Idempotency(rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodPut {
				key := r.Header.Get("Idempotency-Key")
				if key != "" {
					ctx := context.WithValue(r.Context(), idempotencyKey, key)
					exists, err := rdb.Exists(ctx, "idempotent:"+key).Result()
					if err == nil && exists > 0 {
						body, _ := rdb.Get(ctx, "idempotent:"+key).Bytes()
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						w.Write(body)
						return
					}
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func StoreIdempotentResponse(ctx context.Context, rdb *redis.Client, key string, status int, body interface{}) {
	if key == "" {
		return
	}
	data, _ := json.Marshal(body)
	rdb.Set(ctx, "idempotent:"+key, data, 86400e9)
}

func GetIdempotencyKey(ctx context.Context) string {
	if v, ok := ctx.Value(idempotencyKey).(string); ok {
		return v
	}
	return ""
}
