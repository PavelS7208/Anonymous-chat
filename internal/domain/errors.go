package domain

import "errors"

// Ошибки доменного уровня.
// Для сравнения используйте errors.Is().
var (
	// Message parsing & crypto errors

	ErrInvalidPubKeyBase64         = errors.New("invalid pubkey: not valid base64")
	ErrPubKeyInvalidLength         = errors.New("invalid pubkey: must decode to exactly 32 bytes")
	ErrInvalidSignature            = errors.New("invalid signature: not valid base64")
	ErrSignatureInvalidLength      = errors.New("invalid signature: must decode to exactly 64 bytes")
	ErrSignatureVerificationFailed = errors.New("signature verification failed")

	// Room errors

	ErrRoomNameInvalid   = errors.New("invalid room name")
	ErrRoomAlreadyExists = errors.New("room already exists")
	ErrInvalidRoomName   = errors.New("invalid room name")
	ErrRoomFull          = errors.New("room is full")

	// Member errors

	ErrMemberNotFound      = errors.New("member not found")
	ErrMemberAlreadyExists = errors.New("member with this public key already exists")
	ErrMemberIDOverflow    = errors.New("member ID overflow: maximum value reached")

	// Event errors

	ErrInvalidEventType = errors.New("invalid event type")

	// Message errors (business rules)

	ErrMessageEmpty            = errors.New("message content is required")
	ErrMessageTooLong          = errors.New("message exceeds maximum length")
	ErrMessageInvalidUTF8      = errors.New("message contains invalid UTF-8 sequence")
	ErrMessageContainsNewline  = errors.New("message cannot contain newline characters")
	ErrMessageWhitespaceAtEdge = errors.New("message cannot start or end with whitespace")
)
