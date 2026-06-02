package http

import (
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"anonymous-chat/internal/chat"
)

var roomNameRegExp = regexp.MustCompile(chat.RoomNamePattern)

// NewGetHandler возвращает хендлер для GET-запросов
func NewGetHandler(svc *chat.Service, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomName := r.PathValue("roomName")
		if roomName == "" || !roomNameRegExp.MatchString(roomName) {
			http.Error(w, "invalid room name", http.StatusBadRequest)
			return
		}

		// Делаем сессию для обработки
		session, err := svc.NewSession(roomName)
		if err != nil {
			logger.Printf("new session error [%s]: %v", roomName, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer session.Close()

		// Работаем через стрим_writer с Flush
		stream, err := newStreamWriter(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Заголовки для долгоживущего стрима
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		// ВЫводим Приветствие при коннекте (по ТЗ - id подключившегося и его ключ)
		if err := stream.send(session.GetWelcomeString()); err != nil {
			logger.Printf("send welcome error: %v", err)
			return
		}

		// Событие входа отправляем в историю
		joinEvent := chat.NewJoinEvent(time.Now().Unix(), session.GetMemberID())
		session.AddToHistory(joinEvent)

		// Историю подгрузившуюся отправляем себе
		for _, event := range session.GetHistorySnapshot() {
			if err := stream.send(event.String()); err != nil {
				logger.Printf("send history error: %v", err)
				return
			}
		}

		// Уведомление других о своем подключении
		session.Broadcast(joinEvent)

		// непосредственно сам стриминг.
		// Отправляем себе, все что появляется пока не закроемся
		ctx := r.Context()
		for {
			select {
			case event, ok := <-session.GetEvents():
				if !ok {
					return // канал закрыт
				}
				if err := stream.send(event.String()); err != nil {
					logger.Printf("stream send error: %v", err)
					return
				}
			case <-ctx.Done():
				logger.Printf("client disconnected [%s]: %v", roomName, ctx.Err())
				return
			case <-session.Done():
				logger.Printf("session closed by server [%s]", roomName)
				return
			}
		}
	}
}

// NewPostHandler возвращает хендлер для POST-запросов (отправка сообщения).
func NewPostHandler(svc *chat.Service, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomName := r.PathValue("roomName")
		if roomName == "" || !roomNameRegExp.MatchString(roomName) {
			http.Error(w, "invalid room name", http.StatusBadRequest)
			return
		}

		// Чтение тела. 1024 макс длина + оверхеды по максимуму
		raw, err := io.ReadAll(io.LimitReader(r.Body, chat.MaxMessageBytes+256))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		// Корректное закрытие тела с обработкой ошибки
		defer func() {
			if err := r.Body.Close(); err != nil {
				logger.Printf("failed to close request body [%s]: %v", roomName, err)
			}
		}()

		// Парсим тело
		line := strings.TrimSpace(string(raw))
		parts := strings.SplitN(line, " ", 3)
		if len(parts) != 3 {
			http.Error(w, "invalid format: expected '<pub> <sig> <msg>'", http.StatusBadRequest)
			return
		}
		// Получили три составляющих: ключ, подпись,msg
		pubB64, sigB64, msg := parts[0], parts[1], parts[2]

		// Отдаем их сервису на обработку (проверку и отправку сообщения)
		if err := svc.PostMessage(roomName, pubB64, sigB64, msg); err != nil {
			status := httpStatusForError(err)
			logger.Printf("post message error [%s]: %v", roomName, err)
			http.Error(w, err.Error(), status)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
