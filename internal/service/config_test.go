package service

import (
	"testing"
)

// Тест валидации: проверяем граничные условия
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "default config is valid",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "zero maxMessageBytes",
			cfg: Config{
				maxMessageBytes: 0,
				roomNamePattern: `^[a-zA-Z0-9_-]{1,32}$`,
			},
			wantErr: true,
		},
		{
			name: "too large maxMessageBytes",
			cfg: Config{
				maxMessageBytes: limitMessageLength + 1, // > 8KB limit
				roomNamePattern: `^[a-zA-Z0-9_-]{1,32}$`,
			},
			wantErr: true,
		},
		{
			name: "invalid regex pattern",
			cfg: Config{
				maxMessageBytes: 1024,
				roomNamePattern: `[invalid`, // незакрытая скобка
			},
			wantErr: true,
		},
		{
			name: "pattern rejects valid name",
			cfg: Config{
				maxMessageBytes: 1024,
				roomNamePattern: `^[0-9]+$`, // принимает только цифры, "test-room" не пройдёт
			},
			wantErr: true,
		},
		{
			name: "boundary: maxMessageBytes = limitMessageLength",
			cfg: Config{
				maxMessageBytes: limitMessageLength,
				roomNamePattern: `^[a-zA-Z0-9_-]{1,32}$`,
			},
			wantErr: false,
		},
		{
			name: "boundary: maxMessageBytes = 1",
			cfg: Config{
				maxMessageBytes: 1,
				roomNamePattern: `^[a-zA-Z0-9_-]{1,32}$`,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg // копия, чтобы не мутировать оригинал
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Тест опций: проверяем, что With... функции меняют поля
func TestConfig_Options(t *testing.T) {
	cfg := DefaultConfig()

	// Применяем опции вручную (как это делает Configure)
	WithMaxMessageBytes(512)(&cfg)
	WithRoomNamePattern(`^test-.*$`)(&cfg)

	if cfg.maxMessageBytes != 512 {
		t.Errorf("maxMessageBytes = %d, want 512", cfg.maxMessageBytes)
	}
	if cfg.roomNamePattern != `^test-.*$` {
		t.Errorf("roomNamePattern = %q, want %q", cfg.roomNamePattern, `^test-.*$`)
	}
}

// Интеграционный тест: Configure + валидация + применение
func TestConfigure_Integration(t *testing.T) {
	// Сохраняем оригинал для восстановления после теста
	originalCfg := cfg
	t.Cleanup(func() {
		cfg = originalCfg // rollback
	})

	// Успешная конфигурация
	Configure(
		DefaultConfig(),
		WithMaxMessageBytes(2048),
		WithRoomNamePattern(`^room-[a-z]+$`),
	)

	if cfg.maxMessageBytes != 2048 {
		t.Errorf("after Configure: maxMessageBytes = %d, want 2048", cfg.maxMessageBytes)
	}
	if cfg.roomNamePattern != `^room-[a-z]+$` {
		t.Errorf("roomNamePattern = %q, want %q", cfg.roomNamePattern, `^room-[a-z]+$`)
	}
	if cfg.roomNameRe == nil {
		t.Error("after Configure: roomNameRe should be compiled, got nil")
	}
	// Проверка, что регекс действительно работает
	if !cfg.roomNameRe.MatchString("room-test") {
		t.Error("compiled regex should match 'room-test'")
	}
	if cfg.roomNameRe.MatchString("test-room") {
		t.Error("compiled regex should NOT match 'test-room' with pattern ^room-[a-z]+$")
	}
}

// Тест на log. Fatalf при невалидной конфигурации
// В тестах log.Fatalf вызывает os.Exit(1), поэтому проверяем через recover
func TestConfigure_InvalidConfigPanics(t *testing.T) {
	// Перехватываем panic, который вызовет log.Fatalf в тестовой среде
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Configure() did not panic on invalid config")
		}
	}()

	// Эта конфигурация не пройдёт валидацию → log.Fatalf
	Configure(
		Config{
			maxMessageBytes: -1, // invalid: <= 0
			roomNamePattern: `^[a-zA-Z0-9_-]{1,32}$`,
		},
	)
}

// Тест DefaultConfig: проверяем, что дефолты соответствуют константам
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.maxMessageBytes != 1024 {
		t.Errorf("DefaultConfig().maxMessageBytes = %d, want 1024", cfg.maxMessageBytes)
	}
	if cfg.roomNamePattern != `^[a-zA-Z0-9_-]{1,32}$` {
		t.Errorf("DefaultConfig().roomNamePattern = %q, want %q",
			cfg.roomNamePattern, `^[a-zA-Z0-9_-]{1,32}$`)
	}
	// Проверяем, что дефолтный конфиг валиден
	if err := cfg.Validate(); err != nil {
		t.Errorf("DefaultConfig() should be valid, got error: %v", err)
	}
}
