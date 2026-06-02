package http

import (
	"anonymous-chat/internal/chat"
	"errors"
	"net/http"
)

// httpStatusForError маппит ошибки домена на HTTP-статусы
func httpStatusForError(err error) int {
	switch {
	case errors.Is(err, chat.ErrRoomNotFound):
		return http.StatusNotFound
	case errors.Is(err, chat.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, chat.ErrForbidden), errors.Is(err, chat.ErrInvalidSignature):
		return http.StatusForbidden
	case errors.Is(err, chat.ErrInvalidPubKey), errors.Is(err, chat.ErrInvalidMessage):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
