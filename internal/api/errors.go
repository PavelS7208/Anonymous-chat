package api

import (
	"anonymous-chat/internal/service"
	"context"
	"errors"
	"net/http"
)

func httpStatusForError(err error) int {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return http.StatusRequestTimeout
	case errors.Is(err, service.ErrRoomNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrInvalidMessageFormat):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrInvalidCryptoArgs):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrMemberUnauthorized):
		//errors.Is(err, service.ErrForbidden):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
