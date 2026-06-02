// internal/service/errors.go
package service

import "errors"

var (
	ErrRoomNotFound = errors.New("room not found")

	// Операционные ошибки сервиса

	ErrShuttingDown              = errors.New("room manager is shutting down")
	ErrGlobalCreatedRoomReached  = errors.New("service unavailable: max rooms reached")
	ErrGlobalJoinedMemberReached = errors.New("service unavailable: max member in room reached")

	ErrJoinRateLimited = errors.New("join rate limit exceeded")
	ErrPostRateLimited = errors.New("message rate limit exceeded")

	ErrTooManyConnections = errors.New("too many connections")

	ErrSessionNotFound = errors.New("session not found")
	ErrIPMismatch      = errors.New("ip mismatch between Get and Post sessions")
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionIdle     = errors.New("session is idle")
)
