package handler_test

import (
	"context"
	"testing"

	"github.com/pavel/anonymous-chat/internal/service"
)

// mockMessageSender — тестовый мок для service.MessageSender
type mockMessageSender struct {
	sendFunc func(ctx context.Context, req service.SendRequest) error // Кастомная логика
	sendErr  error                                                    // Статическая ошибка для возврата и проверок на нее

	// Фиксация вызовов (для ассертов в тестах)
	calls []service.SendRequest
}

// newMockMessageSender создаёт новый настроенный мок
func newMockMessageSender() *mockMessageSender {
	return &mockMessageSender{
		calls: make([]service.SendRequest, 0, 4), // Пред выделение под типичные тесты
	}
}

// Send реализует интерфейс service.MessageSender
func (m *mockMessageSender) Send(ctx context.Context, req service.SendRequest) error {
	// Фиксируем вызов
	m.calls = append(m.calls, req)

	// Приоритет: если задана кастомная функция — используем её
	if m.sendFunc != nil {
		return m.sendFunc(ctx, req)
	}

	// Иначе возвращаем статическую ошибку (или nil)
	return m.sendErr
}

// --- Хелперы для удобной настройки мока в тестах ---

// WithSendErr настраивает возврат конкретной ошибки
func (m *mockMessageSender) WithSendErr(err error) *mockMessageSender {
	m.sendErr = err
	m.sendFunc = nil // отключаем кастомную логику
	return m
}

// WithSendFunc настраивает кастомную логику обработки запроса
func (m *mockMessageSender) WithSendFunc(fn func(ctx context.Context, req service.SendRequest) error) *mockMessageSender {
	m.sendFunc = fn
	m.sendErr = nil // отключаем статическую ошибку
	return m
}

// Reset очищает историю вызовов (полезно при повторном использовании мока)
func (m *mockMessageSender) Reset() {
	m.calls = nil
	m.sendFunc = nil
	m.sendErr = nil
}

// Calls возвращает скопированный срез вызовов (защита от случайной модификации)
func (m *mockMessageSender) Calls() []service.SendRequest {
	callsCopy := make([]service.SendRequest, len(m.calls))
	copy(callsCopy, m.calls)
	return callsCopy
}

// LastCall возвращает последний вызов (удобно для быстрых ассертов)
func (m *mockMessageSender) LastCall() (service.SendRequest, bool) {
	if len(m.calls) == 0 {
		return service.SendRequest{}, false
	}
	return m.calls[len(m.calls)-1], true
}

// AssertCalled проверяет, что метод был вызван ровно ожидаемое количество раз
func (m *mockMessageSender) AssertCalled(t testing.TB, expectedCount int) {
	t.Helper()
	if len(m.calls) != expectedCount {
		t.Errorf("expected Send() to be called %d times, got %d", expectedCount, len(m.calls))
	}
}

// AssertNotCalled проверяет, что метод не вызывался
func (m *mockMessageSender) AssertNotCalled(t testing.TB) {
	t.Helper()
	m.AssertCalled(t, 0)
}
