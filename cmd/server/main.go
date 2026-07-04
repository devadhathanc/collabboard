package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/devadhathan/collabboard/internal/auth"
	"github.com/devadhathan/collabboard/internal/board"
	"github.com/devadhathan/collabboard/internal/config"
	"github.com/devadhathan/collabboard/internal/db"
	"github.com/devadhathan/collabboard/internal/redisc"
	"github.com/devadhathan/collabboard/internal/ws"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	rc, err := redisc.NewClient(ctx, cfg.Redis.URL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rc.Close()

	hub := ws.NewHub(rc)

	refreshStore := db.NewRefreshTokenStore(pool)
	denylist := auth.NewRedisDenylist(rc.Raw())
	authSvc, err := auth.NewService(
		time.Duration(cfg.Auth.AccessTokenTTL)*time.Second,
		refreshStore,
		denylist,
	)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}

	auditLogger := auth.NewAuditLogger(pool)
	rateLimiter := auth.NewRateLimiter(rc.Raw(), cfg.Auth.RateLimitPerIP, cfg.Auth.RateLimitPerAcct)

	taskRepo := db.NewTaskRepo(pool)
	boardSvc := board.NewService(taskRepo, rc)
	boardHandler := board.NewHandler(boardSvc, hub)

	authMW := auth.Middleware(authSvc)
	auditMW := auth.AuditMiddleware(auditLogger)
	rlMW := rateLimiter.Middleware(false)

	mux := http.NewServeMux()

	auth.NewHandler(authSvc).RegisterRoutes(mux)
	mux.HandleFunc("GET /ws", ws.ServeWS(hub))

	protected := http.NewServeMux()
	boardHandler.RegisterRoutes(protected)

	mux.Handle("/boards/", rlMW(auditMW(authMW(protected))))

	addr := cfg.Server.Host + ":" + strconv.Itoa(cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	hub.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
}
