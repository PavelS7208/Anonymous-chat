package chat

import (
	"errors"
	"fmt"
	"log"
)

// Config хранит параметры инициализации пакета.
type Config struct {
	initialHistoryLength    int // Длина списка последних сообщений для нового участника
	maxHistoryStorageLength int
	initialHistoryCap       int
	maxMemberID             MemberID // Лимит индекса memberID внутри комнаты
	eventChannelBuf         int
}

var cfg Config

// Option — функциональная опция для настройки конфигурации
type Option func(*Config)

// DefaultConfig возвращает конфигурацию со значениями по умолчанию
// Все дефолты объявлены тут
func DefaultConfig() Config {
	return Config{
		initialHistoryLength:    10,
		maxHistoryStorageLength: 1000,
		initialHistoryCap:       2000,
		maxMemberID:             MemberID((1 << 63) - 1),
		eventChannelBuf:         64,
	}
}

// --------- для вызова из вне ------------

// WithInitialHistoryLength задает длину истории для нового участника
func WithInitialHistoryLength(n int) Option {
	return func(c *Config) { c.initialHistoryLength = n }
}

// WithMaxHistoryStorageLength задает лимит хранения истории
// Автоматически обновляет initialHistoryCap, если он не был задан явно
func WithMaxHistoryStorageLength(n int) Option {
	return func(c *Config) {
		c.maxHistoryStorageLength = n
		// Авто-расчёт cap как * 2
		if c.initialHistoryCap == 0 || c.initialHistoryCap == DefaultConfig().initialHistoryCap {
			c.initialHistoryCap = n * 2
		}
	}
}

// WithInitialHistoryCap явно задает capacity истории
// Это если нужно переопределить авто-расчёт cap
func WithInitialHistoryCap(n int) Option {
	return func(c *Config) { c.initialHistoryCap = n }
}

// WithMaxMemberID задает максимальный лимит memberID
func WithMaxMemberID(id MemberID) Option {
	return func(c *Config) { c.maxMemberID = id }
}

// WithEventChannelBuf задает размер буфера канала событий
func WithEventChannelBuf(n int) Option {
	return func(c *Config) { c.eventChannelBuf = n }
}

// Validate проверяет корректность конфигурации.

func (c *Config) Validate() error {
	var errs []error

	if c.initialHistoryLength <= 0 {
		errs = append(errs, fmt.Errorf("initialHistoryLength must be > 0, got: %d", c.initialHistoryLength))
	}
	if c.maxHistoryStorageLength <= 0 {
		errs = append(errs, fmt.Errorf("maxHistoryStorageLength must be > 0, got: %d", c.maxHistoryStorageLength))
	}
	if c.initialHistoryLength > c.maxHistoryStorageLength {
		errs = append(errs, fmt.Errorf("initialHistoryLength (%d) cannot exceed maxHistoryStorageLength (%d)",
			c.initialHistoryLength, c.maxHistoryStorageLength))
	}
	if c.initialHistoryCap < c.maxHistoryStorageLength {
		errs = append(errs, fmt.Errorf("initialHistoryCap (%d) must be >= maxHistoryStorageLength (%d)",
			c.initialHistoryCap, c.maxHistoryStorageLength))
	}
	if c.maxMemberID <= 0 {
		errs = append(errs, fmt.Errorf("maxMemberID must be > 0, got: %d", c.maxMemberID))
	}
	if c.eventChannelBuf <= 0 {
		errs = append(errs, fmt.Errorf("eventChannelBuf must be > 0, got: %d", c.eventChannelBuf))
	}

	if len(errs) > 0 {
		return fmt.Errorf("chat config validation failed: %w", errors.Join(errs...))
	}
	return nil
}

// apply применяет валидную конфигурацию
func (c *Config) apply() {
	cfg = *c
}

// Configure — точка входа для настройки пакета
// принимает дефолтную конфигурацию и набор опций
// паникует при ошибке валидации, чтобы остановить некорректный запуск
// Стандартный шаблон для опций
func Configure(base Config, opts ...Option) {
	// Применяем опции к базовой конфигурации
	for _, opt := range opts {
		opt(&base)
	}

	// Валидация перед применением
	if err := base.Validate(); err != nil {
		log.Fatalf("[chat] конфигурация невалидна: %v", err)
	}

	// Применяем к внутреннему состоянию
	base.apply()
}
