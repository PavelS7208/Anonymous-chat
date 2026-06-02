package service

import (
	"anonymous-chat/internal/crypto"
	"errors"
	"fmt"
)

var (
	ErrRoomNotFound         = errors.New("room not found")
	ErrMemberUnauthorized   = errors.New("unauthorized member")
	ErrInvalidMessageFormat = errors.New("invalid message format: expected '<pubKey> <sig> <msg>'")
	ErrInvalidCryptoArgs    = errors.New("invalid crypto arguments: <sig> or <pubKey>")
)

// mapCryptoError преобразует ошибки криптографии в бизнес-ошибки
func mapCryptoError(err error) error {
	switch {
	case errors.Is(err, crypto.ErrInvalidPublicKey),
		errors.Is(err, crypto.ErrInvalidSignature),
		errors.Is(err, crypto.ErrSignatureVerify):

		return ErrInvalidCryptoArgs
	default:
		return fmt.Errorf("crypto: %w", err)
	}
}
