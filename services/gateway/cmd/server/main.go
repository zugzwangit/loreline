package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/loreline/loreline/services/gateway/internal/platform"
)

func main() {
	addr := env("LORELINE_GATEWAY_ADDR", ":8080")
	ai := env("LORELINE_AI_URL", "http://localhost:8090")
	server := &http.Server{Addr: addr, Handler: platform.NewHandler(platform.NewStore(), ai), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("Loreline gateway listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
