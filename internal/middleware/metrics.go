package middleware

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

type Metrics struct {
	ActiveConns    int64
	TotalRequests  int64
	ConflictCount  int64
	AuthFailures   int64
	WSConnections  int64
}

var GlobalMetrics = &Metrics{}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&GlobalMetrics.TotalRequests, 1)
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		if rw.status == 409 {
			atomic.AddInt64(&GlobalMetrics.ConflictCount, 1)
		}
		if rw.status == 401 {
			atomic.AddInt64(&GlobalMetrics.AuthFailures, 1)
		}

		log.Printf("metric method=%s path=%s status=%d duration=%s",
			r.Method, r.URL.Path, rw.status, duration)
	})
}

func IncWSConnections()  { atomic.AddInt64(&GlobalMetrics.WSConnections, 1) }
func DecWSConnections()  { atomic.AddInt64(&GlobalMetrics.WSConnections, -1) }
func IncActiveConns()    { atomic.AddInt64(&GlobalMetrics.ActiveConns, 1) }
func DecActiveConns()    { atomic.AddInt64(&GlobalMetrics.ActiveConns, -1) }
