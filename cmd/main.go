package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/msniranjan18/online-clipboard/pkg/hub"
	"github.com/msniranjan18/online-clipboard/pkg/routes"
	"github.com/msniranjan18/online-clipboard/pkg/store"
)

func main() {
	// 1. Initialize Storage (Postgres & Redis)
	pgConn := os.Getenv("DATABASE_URL")
	if pgConn == "" {
		pgConn = "postgres://user:pass@localhost:5432/clipdb?sslmode=disable"
	}
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	storage, err := store.NewStore(pgConn, redisAddr)
	if err != nil {
		log.Fatalf("Failed to connect to storage: %v", err)
	}

	// Run cleanup every 1 hour, deleting data older than 2 hours
	// Use a goroutine so it doesn't block the server startup
	go storage.StartCleanupWorker(1*time.Hour, 24*time.Hour)

	// 2. Initialize Hub & Background Workers
	wsHub := hub.NewHub(storage)
	go wsHub.Run()
	go wsHub.ListenToRedis()

	// 3. Initialize Router from the routes package
	router := routes.NewRouter(wsHub)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default for local dev
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}
	log.Printf("Server starting on :%s...", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
