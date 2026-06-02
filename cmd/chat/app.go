package main

import (
	"anonymous-chat/internal/api"
	"anonymous-chat/internal/service"
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// NewAnonymousChatApp создаёт и конфигурирует приложение через функциональные опции
func NewAnonymousChatApp(opts ...AppOption) *AnonymousChatApp {
	// Базовая конфигурация
	cfg := DefaultAppConfig()

	// Применяем пользовательские опции
	for _, opt := range opts {
		opt(cfg)
	}

	// Валидация (panic при ошибке, так как это конструктор)
	if err := cfg.Validate(); err != nil {
		panic(err)
	}

	svc := service.NewAnonymousChat()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{roomName}", api.NewGetHandler(svc, cfg.Logger))
	mux.HandleFunc("POST /{roomName}", api.NewPostHandler(svc, cfg.Logger))

	httpSrv := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           mux,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	return &AnonymousChatApp{
		chatBackender: svc,
		logger:        cfg.Logger,
		addr:          cfg.ServerAddr,
		httpSrv:       httpSrv,
	}
}

// Run запускает сервер и блокируется до сигнала остановки
func (a *AnonymousChatApp) Run() error {
	serverErrors := make(chan error, 1)
	go func() {
		a.logger.Printf("HTTP server starting on %s", a.addr)
		serverErrors <- a.httpSrv.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return err // ошибка сервера
	case sig := <-shutdown:
		a.logger.Printf("Received signal %s, starting graceful shutdown", sig)
		return nil // нормальный путь завершения
	}
}

// Close освобождает ресурсы в правильном порядке
func (a *AnonymousChatApp) Close(ctx context.Context) error {
	// 1. Останавливаем HTTP — перестаем принимать новые соединения
	a.logger.Println("Shutting down HTTP server...")
	if err := a.httpSrv.Shutdown(ctx); err != nil {
		a.logger.Printf("HTTP shutdown warning: %v", err)
	}

	// 2. Закрываем бизнес-логику — комнаты и участники
	a.logger.Println("Closing chat service...")
	if err := a.chatBackender.Close(); err != nil {
		a.logger.Printf("chatBackender close warning: %v", err)
	}

	a.logger.Println("Server stopped gracefully")
	return nil
}
