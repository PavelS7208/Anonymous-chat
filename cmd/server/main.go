package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pavel/anonymous-chat/internal/app"
	"github.com/pavel/anonymous-chat/internal/config"
)

func main() {

	// Загрузка инфраструктуры из env + дефолтов
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load error: %v", err)
	}
	// Гарантируем закрытие файла (Os-Exit-ов нет далее)
	if cfg.LogFile != nil {
		defer cfg.LogFile.Close()
	}

	logger := slog.New(slog.NewJSONHandler(cfg.LogWriter, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	// Создание приложения
	application, err := app.NewAnonymousChatApp(cfg, logger)
	if err != nil {
		logger.Error("app creation error: %v", err)
		return
	}

	// Контекст для управления всем приложением
	// Отмена этого контекста остановит все фоновые горутины
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Запуск: один вызов, всё внутри
	// При штатном завершении вернёт nil, при ошибке — err
	if err := application.Run(ctx); err != nil {
		logger.Error("app error: %v", err)
		return
	}

	logger.Info("server stopped gracefully")
}
