package handler_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handler2 "github.com/pavel/anonymous-chat/internal/adapters/handler"
	"github.com/pavel/anonymous-chat/internal/domain"
	"github.com/pavel/anonymous-chat/internal/service"
)

// Тест на положительную ветку
func TestPostHandler_Success(t *testing.T) {
	sender := newMockMessageSender()

	handler := handler2.NewPostHandler(sender, newTestLogger())

	body := []byte("pk sig hello world\n")
	req := httptest.NewRequest(http.MethodPost, "/test_room", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	req.RemoteAddr = "127.0.0.1:12345"

	rr := httptest.NewRecorder()
	r := newChiPostTestRouter(handler)
	r.ServeHTTP(rr, req)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf(
			"got status %d, want %d\nbody: %q\nheaders: %v",
			status, http.StatusNoContent,
			rr.Body.String(),
			rr.Header(),
		)
	}

	sender.AssertCalled(t, 1)
	lastCall, ok := sender.LastCall()
	if !ok {
		t.Fatal("expected at least one call")
	}
	if lastCall.RoomName != "test_room" {
		t.Errorf("roomName = %q, want %q", lastCall.RoomName, "test_room")
	}
	if string(lastCall.Message) != "hello world" {
		t.Errorf("message = %q, want %q", lastCall.Message, "hello world")
	}
}

// Тест на то что отловится превышение размера сообщения
func TestPostHandler_MaxBytesLimit(t *testing.T) {
	sender := newMockMessageSender()

	handler := handler2.NewPostHandler(sender, newTestLogger())

	// Создаём роутер и регистрируем хендлер с паттерном
	// Обёртываем ВЕСЬ роутер в MaxBytesHandler (как в продакшене)
	r := newChiPostTestRouter(handler)
	maxBytesHandler := http.MaxBytesHandler(r, 4096)

	// Задаем тело больше лимита: 4100 > 4096
	body := bytes.Repeat([]byte("x"), 4100)
	req := httptest.NewRequest(http.MethodPost, "/room_test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	req.RemoteAddr = "127.0.0.1:12345"

	rr := httptest.NewRecorder()
	// Прогоняем через обёрнутый роутер
	maxBytesHandler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusRequestEntityTooLarge {
		t.Errorf(
			"got status %d, want %d\nbody: %q",
			status, http.StatusRequestEntityTooLarge,
			rr.Body.String(),
		)
	}

	// убеждаемся, что сервис не был вызван, отбило на этапе проверок
	sender.AssertNotCalled(t)
}

// Тест что придет ошибка http. StatusUnauthorized и обезличинный ответ (401 Unauthorized),
// чтобы не дать возможность перебирать
// Если на входе неверная подпись в теле POST
func TestPostHandler_InvalidSignature(t *testing.T) {
	sender := newMockMessageSender().WithSendErr(domain.ErrInvalidSignature)

	handler := handler2.NewPostHandler(sender, newTestLogger())

	body := []byte("pk badsig msg\n")
	req := httptest.NewRequest(http.MethodPost, "/room_test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()
	r := newChiPostTestRouter(handler)
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", status, http.StatusUnauthorized)
	}
	if !strings.Contains(rr.Body.String(), "401 Unauthorized") {
		t.Errorf("response body = %q, want substring %q", rr.Body.String(), "401 Unauthorized")
	}
}

// Тест на реакцию ошибок бизнес логики по длине сообщения
func TestPostHandler_CustomLogic(t *testing.T) {
	sender := newMockMessageSender().WithSendFunc(func(ctx context.Context, req service.SendRequest) error {
		// Эмулируем бизнес-логику: отклоняем сообщения короче 5 символов
		if len(req.Message) < 5 {
			return errors.New("message too short")
		}
		return nil
	})

	handler := handler2.NewPostHandler(sender, newTestLogger())

	// Короткое сообщение -> 500 (неизвестная ошибка сервиса)
	body := []byte("pk sig hi\n")
	req := httptest.NewRequest(http.MethodPost, "/room_test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()
	r := newChiPostTestRouter(handler)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("short msg: got %d, want 500", rr.Code)
	}

	// Длинное сообщение -> 204
	sender.Reset() // очищаем историю
	body = []byte("pk sig hello world\n")
	req = httptest.NewRequest(http.MethodPost, "/room_test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	rr = httptest.NewRecorder()
	r = newChiPostTestRouter(handler)
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("long msg: got %d, want 204", rr.Code)
	}
}

// Тест проверка реакции на RateLimit ошибки в слое сервиса (429 с заголовком)
func TestPostHandler_RateLimited(t *testing.T) {
	sender := newMockMessageSender().WithSendErr(service.ErrPostRateLimited)

	handler := handler2.NewPostHandler(sender, newTestLogger())

	body := []byte("pk sig msg\n")
	req := httptest.NewRequest(http.MethodPost, "/room_test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()
	r := newChiPostTestRouter(handler)
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("got status %d, want %d", status, http.StatusTooManyRequests)
	}
	if retry := rr.Header().Get("Retry-After"); retry != "60" {
		t.Errorf("Retry-After = %q, want %q", retry, "60")
	}
}
