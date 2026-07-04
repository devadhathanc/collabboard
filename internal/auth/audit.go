package auth

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLogger struct {
	pool *pgxpool.Pool
}

func NewAuditLogger(pool *pgxpool.Pool) *AuditLogger {
	return &AuditLogger{pool: pool}
}

func (a *AuditLogger) Log(ctx context.Context, actorID, action, resourceType string, resourceID, ip string) error {
	_, err := a.pool.Exec(ctx,
		`INSERT INTO audit_log (actor_id, action, resource_type, resource_id, ip, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		actorID, action, resourceType, resourceID, ip, time.Now().UTC(),
	)
	if err != nil {
		log.Printf("audit log error: %v", err)
	}
	return err
}

func AuditMiddleware(logger *AuditLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPatch ||
				r.Method == http.MethodPut || r.Method == http.MethodDelete {
				claims := GetClaims(r.Context())
				if claims != nil {
					ip := r.RemoteAddr
					if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
						ip = fwd
					}
					go logger.Log(context.Background(), claims.UserID, r.Method, r.URL.Path, "", ip)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
