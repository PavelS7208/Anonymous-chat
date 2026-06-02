package main

import (
	"anonymous-chat/internal/chat"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	chathttp "anonymous-chat/internal/http"
)


func main() {
	chatSvc := chat.NewService() // параметры по умолчанию

	// Логгер
	logger := log.New(os.Stdout, "[chat] ", log.LstdFlags|log.Lshortfile)

	mux := http.NewServeMux()

	// Регистрация хендлеров
	// /{roomName} — roomName извлекается PathValue()
	mux.HandleFunc("GET /{roomName}", chathttp.NewGetHandler(chatSvc, logger))
	mux.HandleFunc("POST /{roomName}", chathttp.NewPostHandler(chatSvc, logger))

	addr := ":8080"
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("HTTP server starting on %s", addr)
		serverErrors <- httpSrv.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("Server failed to start: %v", err)
	case sig := <-shutdown:
		log.Printf("Received signal %s, initiating graceful shutdown...", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	log.Println("Waiting for active connections to finish...")
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Fatalf("Graceful shutdown failed or timed out: %v", err)
	}
	log.Println("Server stopped gracefully")

}
