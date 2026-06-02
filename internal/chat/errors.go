package chat

import "errors"

var (
	ErrRoomNotFound     = errors.New("room not found")
	ErrForbidden        = errors.New("forbidden: invalid signature")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrUnauthorized     = errors.New("unauthorized member")
	ErrInvalidPubKey    = errors.New("invalid public key")
	ErrInvalidMessage   = errors.New("invalid message")
)
