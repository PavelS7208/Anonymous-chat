package chat

import "errors"

var (
	ErrInvalidRoomName         = errors.New("invalid room name")
	ErrInvalidMessage          = errors.New("invalid message")
	ErrMemberIdOverflow        = errors.New("member ID overflow")
	errMemberAlreadyRegistered = errors.New("member with that PublicB64Key or ID already registered")
)
