package main

import (
	"anonymous-chat/internal/chat"
	"anonymous-chat/internal/service"
	"fmt"
	"log"
	"os"
	"time"
)

// AppConfig — полная конфигурация приложения
type AppConfig struct {
	ServerAddr        string
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration

	ChatConfig    chat.Config
	ServiceConfig service.Config

	Logger *log.Logger
}

var appConfig AppConfig

// DefaultAppConfig возвращает конфигурацию со значениями по умолчанию
func DefaultAppConfig() *AppConfig {
	return &AppConfig{
		ServerAddr:        ":8080",
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		ChatConfig:        chat.DefaultConfig(),
		ServiceConfig:     service.DefaultConfig(),
		Logger:            log.New(os.Stdout, "[chat] ", log.LstdFlags|log.Lshortfile),
	}
}

// Validate проверяет корректность конфигурации
func (cfg *AppConfig) Validate() error {
	if cfg.ServerAddr == "" {
		return errConfig("server address cannot be empty")
	}
	if cfg.ReadHeaderTimeout <= 0 {
		return errConfig("read header timeout must be positive")
	}
	if cfg.IdleTimeout <= 0 {
		return errConfig("idle timeout must be positive")
	}
	if err := cfg.ChatConfig.Validate(); err != nil {
		return errConfig("invalid chat config: %w", err)
	}
	if err := cfg.ServiceConfig.Validate(); err != nil {
		return errConfig("invalid service config: %w", err)
	}
	return nil
}

// errConfig — вспомогательная функция для ошибок конфигурации
func errConfig(format string, args ...interface{}) error {
	return fmt.Errorf("config error: "+format, args...)
}

// ------------ Functional Options Pattern ------------

// AppOption — функция, изменяющая AppConfig
type AppOption func(*AppConfig)

// WithServerAddr устанавливает адрес сервера
func WithServerAddr(addr string) AppOption {
	return func(cfg *AppConfig) {
		cfg.ServerAddr = addr
	}
}

// WithTimeouts устанавливает тайм-ауты сервера
func WithTimeouts(readHeader, idle time.Duration) AppOption {
	return func(cfg *AppConfig) {
		cfg.ReadHeaderTimeout = readHeader
		cfg.IdleTimeout = idle
	}
}

// WithChatConfig применяет опции к глобальной конфигурации пакета chat
// Configure принимает Config по значению, применяет опции и обновляет package-level переменную
// Возвращаемое значение отсутствует, поэтому присваивание не требуется
func WithChatConfig(opts ...chat.Option) AppOption {
	return func(cfg *AppConfig) {
		chat.Configure(cfg.ChatConfig, opts...)
	}
}

// WithServiceConfig применяет опции к глобальной конфигурации пакета service
func WithServiceConfig(opts ...service.Option) AppOption {
	return func(cfg *AppConfig) {
		service.Configure(cfg.ServiceConfig, opts...)
	}
}

// WithLogger устанавливает кастомный логгер
func WithLogger(logger *log.Logger) AppOption {
	return func(cfg *AppConfig) {
		if logger != nil {
			cfg.Logger = logger
		}
	}
}
