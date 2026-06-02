package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/pavel/anonymous-chat/internal/adapters/middleware"

	"github.com/pavel/anonymous-chat/internal/adapters/handler"
	"github.com/pavel/anonymous-chat/internal/config"
	"github.com/pavel/anonymous-chat/internal/domain/crypto"
	"github.com/pavel/anonymous-chat/internal/service"
)

// ChatBackender — фасад-интерфейс чата (объединяет оркестрацию комнат и приём сообщений)
// Объединяет RoomManager и MessageSender, так как в данном приложении
// они всегда используются вместе и реализуются одним компонентом.
// Для тестирования можно мокать только используемые методы.
type ChatBackender interface {
	service.RoomManager
	service.MessageSender
}

// ChatBackenderLifecycle — контракт для управления жизненным циклом сервиса.
// Не является частью бизнес-интерфейса ChatBackender
type ChatBackenderLifecycle interface {
	Start(ctx context.Context) error
	Close() error
}

// AnonymousChatApp — корневой тип приложения.
// Инкапсулирует все зависимости и управляет жизненным циклом
type AnonymousChatApp struct {
	chatSvc   ChatBackender
	lifecycle ChatBackenderLifecycle
	httpSrv   *http.Server
	logger    *slog.Logger
	addr      string

	cfg config.Config
}

// NewAnonymousChatApp создаёт и конфигурирует приложение.
// Возвращает ошибку при неверной конфигурации.
// Фоновые процессы НЕ запускаются — вызовите Run(ctx) для старта
func NewAnonymousChatApp(cfg config.Config, logger *slog.Logger) (*AnonymousChatApp, error) {

	cryptoProvider := crypto.NewEd25519Provider()

	// Создаём сервис + контроллер жизненного цикла
	// roomManager полностью инкапсулирован внутри service
	chatSvc, lifecycle := service.NewChatService(cfg.RoomService, cryptoProvider, logger)

	// Настраиваем chi-роутер
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.PostBodyLimits(cfg.MaxPostBodyBytes)) // Защита от огромных строк в POST
	r.Use(middleware.GlobalLimiter(cfg.MaxRequestPerSec))  // Защита от огромных атак массовыми запросами
	//r.Use(chimw.Logger)
	r.Use(middleware.RequestLogger(logger))
	r.Use(chimw.Recoverer)

	// Регистрация хендлеров (получают только интерфейсы)
	// GetHandler не зависит от config — только от сервиса и логгера
	getHandler := handler.NewGetHandler(chatSvc, logger)
	// PostHandler получает только нужный транспортный параметр
	postHandler := handler.NewPostHandler(chatSvc, logger)

	r.Get("/{room_name}", getHandler.ServeHTTP)
	r.Post("/{room_name}", postHandler.ServeHTTP)

	// Health check для orchestration
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Эндпоинт смены уровня лога на лету
	r.Put("/debug/loglevel", func(w http.ResponseWriter, r *http.Request) {
		level := r.URL.Query().Get("level")
		if err := cfg.LogLevel.UnmarshalText([]byte(level)); err != nil {
			http.Error(w, "unknown level: "+level, http.StatusBadRequest)
			return
		}
		logger.Info("debug log level set to " + level)
	})

	// HTTP-сервер с тайм-аутами
	// WriteTimeout=0 отключён для поддержки долгого стриминга
	httpSrv := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           r,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout, // защита от Slowloris на заголовках
		WriteTimeout:      0,                     // ставим 0
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    4 * 1024,
	}

	return &AnonymousChatApp{
		chatSvc:   chatSvc,
		lifecycle: lifecycle,
		httpSrv:   httpSrv,
		logger:    logger,
		addr:      cfg.ServerAddr,
		cfg:       cfg,
	}, nil
}

// Run запускает ВСЁ приложение: фоновые процессы + HTTP-сервер.
// Блокируется до завершения. Возвращает ошибку только при реальном сбое.
// Принимает контекст для корректного graceful shutdown.
func (a *AnonymousChatApp) Run(ctx context.Context) error {
	// Запускаем фоновые процессы (внутренний контекст — дочерний от ctx)
	bgCtx, bgCancel := context.WithCancel(ctx)
	if err := a.lifecycle.Start(bgCtx); err != nil {
		bgCancel()
		_ = a.lifecycle.Close()
		return err
	}

	// Запускаем HTTP-сервер в горутине
	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("server starting", "addr", a.addr)
		errCh <- a.httpSrv.ListenAndServe()
	}()

	// Ждём сигнала или ошибки сервера
	select {
	case <-ctx.Done():
		// Graceful shutdown по сигналу
		a.logger.Info("shutdown signal received")
		bgCancel() // останавливаем фоновые горутины

		// Даём активным соединениям время завершиться
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
		defer cancel()
		if err := a.httpSrv.Shutdown(shutdownCtx); err != nil {
			a.logger.Warn("http shutdown warning", "err", err)
		}
		_ = a.lifecycle.Close() // закрываем комнаты, ресурсы
		return nil

	case err := <-errCh:
		bgCancel()
		_ = a.lifecycle.Close()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	}
}
