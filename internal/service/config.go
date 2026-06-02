package service

import (
	"fmt"
	"log"
	"regexp"
)

// Config хранит параметры сервисной части (бизнесовые)
type Config struct {
	maxMessageBytes int
	roomNamePattern string
	roomNameRe      *regexp.Regexp // Кэшированная компиляция
}

const limitMessageLength = 8 * 1024

var cfg Config

// Option — функциональная опция для настройки.
type Option func(*Config)

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		maxMessageBytes: 1024,
		roomNamePattern: `^[a-zA-Z0-9_-]{1,32}$`,
	}
}

// --- опции ----

func WithMaxMessageBytes(n int) Option {
	return func(c *Config) {
		c.maxMessageBytes = n
	}
}

func WithRoomNamePattern(pattern string) Option {
	return func(c *Config) { c.roomNamePattern = pattern }
}

// Validate проверяет корректность конфигурации.
func (c *Config) Validate() error {
	if c.maxMessageBytes <= 0 || c.maxMessageBytes > limitMessageLength {
		return fmt.Errorf("maxMessageBytes must be in range [1, %d], got: %d", limitMessageLength, c.maxMessageBytes)
	}
	re, err := regexp.Compile(c.roomNamePattern)
	if err != nil {
		return fmt.Errorf("invalid roomNamePattern regex: %w", err)
	}
	c.roomNameRe = re // кэшируем для дальнейшего использования
	// Опциональная проверка: паттерн должен принимать хотя бы что-то разумное
	if !re.MatchString("test-room") {
		return fmt.Errorf("roomNamePattern rejects valid example 'test-room'")
	}
	return nil
}

// apply применяет конфигурацию
func (c *Config) apply() {
	cfg = *c // где cfg — ваша глобальная переменная в service
}

// Configure — точка входа: опции + валидация + применение
func Configure(base Config, opts ...Option) {
	for _, opt := range opts {
		opt(&base)
	}
	if err := base.Validate(); err != nil {
		log.Fatalf("[service] configuration invalid: %v", err)
	}
	base.apply()
}
