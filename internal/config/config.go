package config

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/pavel/anonymous-chat/internal/service"
)

// Config — инфраструктурная конфигурация приложения.
// Загружается один раз при старте из env + дефолтов в коде.
// Не содержит бизнес-правил (те живут в domain.Get*()).
type Config struct {
	ServerAddr        string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration // 0 = отключен (для стриминга)
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration

	SessionTTL time.Duration // Время жизни сессии. Далее считаем зависшей

	MaxPostBodyBytes int // Защита от очень большого поля в POST. Дефолт 2048
	MaxRequestPerSec int // Защита от огромного кол-ва запросов. Дефолт 100 в секунду

	LogLevel    *slog.LevelVar
	LogWriter   io.Writer
	LogFile     *os.File // открытый дескриптор, nil если только stdout
	LogFilePath string   // Путь к файлу — строка для дефолта и env

	RoomService service.RoomServiceConfig
}

// Load загружает конфигурацию с приоритетами:
// 1. Дефолты в коде
// 2. Environment variables (переопределяют дефолты)
// Возвращает ошибку при критических проблемах валидации.
func Load() (Config, error) {
	cfg := Config{
		// Дефолты
		ServerAddr:        ":8080",
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      0, // отключен для поддержки долгого стриминга
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		ShutdownTimeout:   10 * time.Second,

		MaxPostBodyBytes: 2048,
		MaxRequestPerSec: 100,

		LogLevel:    new(slog.LevelVar),
		LogFilePath: `.\chat.log`, // дефолт под Windows
	}

	// Загрузка из env
	if v := os.Getenv("SERVER_ADDR"); v != "" {
		cfg.ServerAddr = v
	}
	if v := os.Getenv("READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ReadTimeout = d
		}
	}
	if v := os.Getenv("READ_HEADER_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ReadHeaderTimeout = d
		}
	}
	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ShutdownTimeout = d
		}
	}
	if v := os.Getenv("MAX_POST_BYTES"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			cfg.MaxPostBodyBytes = d
		}
	}
	if v := os.Getenv("MAX_REQUEST_PER_SEC"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			cfg.MaxRequestPerSec = d
		}
	}

	cfg.LogLevel.Set(slog.LevelInfo) // дефолт уровень

	// LOG_LEVEL из env переопределяет дефолт
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(v)); err != nil {
			return Config{}, fmt.Errorf("unknown log level: %s", v)
		}
	}

	// LOG_FILE из env переопределяет дефолт пути
	if v := os.Getenv("LOG_FILE"); v != "" {
		cfg.LogFilePath = v
	}

	// Открываем файл если путь задан
	if cfg.LogFilePath != "" {
		f, err := os.OpenFile(cfg.LogFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return Config{}, fmt.Errorf("open log file %s: %w", cfg.LogFilePath, err)
		}
		cfg.LogFile = f
		cfg.LogWriter = io.MultiWriter(os.Stdout, f)
	} else {
		cfg.LogWriter = os.Stdout
	}

	// Валидация
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate проверяет критические параметры.
func (c *Config) Validate() error {
	if c.ReadTimeout < 0 {
		return errors.New("ReadTimeout cannot be negative")
	}
	return nil
}
