package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"ideaboard/board"
	"ideaboard/gemini"
	"ideaboard/handlers"
	"ideaboard/ws"
)

func main() {
	// Load .env file if present (before reading os.Getenv).
	loadEnv(".env")

	port := getEnv("PORT", "8090")
	geminiKey := getEnv("GEMINI_API_KEY", "")

	store := board.NewStore()
	hub := ws.NewHub()
	geminiClient := gemini.NewClient(geminiKey)

	ideasH := handlers.NewIdeasHandler(store, hub)
	synthesizeH := handlers.NewSynthesizeHandler(store, geminiClient, hub)

	mux := http.NewServeMux()

	// CORS middleware.
	handler := corsMiddleware(mux)

	// REST endpoints.
	mux.HandleFunc("GET /api/boards/{boardID}/ideas", ideasH.List)
	mux.HandleFunc("POST /api/boards/{boardID}/ideas", ideasH.Submit)
	mux.HandleFunc("POST /api/boards/{boardID}/ideas/synthesize", synthesizeH.Synthesize)

	// WebSocket.
	mux.HandleFunc("GET /ws", hub.ServeWS)

	// Health check.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("IdeaBoard server listening on :%s", port)
	log.Printf("Gemini synthesis enabled: %v", geminiClient.Enabled())

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}

	_ = context.Background()
}

// loadEnv reads KEY=VALUE pairs from a file and sets them as env vars.
// Lines starting with # are ignored. Already-set env vars are not overwritten.
func loadEnv(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return // .env is optional
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Don't override vars already set in the environment.
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
