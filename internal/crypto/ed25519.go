package crypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
)

var (
	ErrInvalidPublicKey = errors.New("invalid public key")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrSignatureVerify  = errors.New("signature verification failed")
)

// GenerateKeyPair генерирует пару ключей Ed25519 по RFC 8032.
// Возвращает seed (32 байта) и публичный ключ (32 байта).
func GenerateKeyPair() (privateSeed, publicKey []byte, err error) {
	// GenerateKey возвращает (publicKey, PrivateKey, error)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ed25519 key pair: %w", err)
	}

	// ed25519.PrivateKey — это []byte длиной 64 байта
	// [0:32] — privateSeed (приватное семя), [32:64] — публичный ключ (для оптимизации) - не используем
	privateSeed = privateKey[:32]

	return privateSeed, publicKey, nil
}

// EncodeBase64 кодирует байты в стандартный base64 (с padding `=`).
func EncodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// DecodeBase64 декодирует base64-строку.
func DecodeBase64(s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}
	return b, nil
}

// validateSizes — проверяем размеры на корректность
func validateSizes(publicKey, sig []byte) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return ErrInvalidPublicKey
	}
	if len(sig) != ed25519.SignatureSize {
		return ErrInvalidSignature
	}
	return nil
}

// Verify - проверка корректности подписи
func Verify(pubKey, sig, message []byte) bool {
	return validateSizes(pubKey, sig) == nil &&
		ed25519.Verify(ed25519.PublicKey(pubKey), message, sig)
}

// VerifyMessageBase64 - Более комплексная проверка для base64
func VerifyMessageBase64(pubKeyB64, sigB64 string, message []byte) error {
	pubBytes, err := DecodeBase64(pubKeyB64)
	if err != nil {
		return ErrInvalidPublicKey
	}
	sigBytes, err := DecodeBase64(sigB64)
	if err != nil {
		return ErrInvalidSignature
	}
	if err := validateSizes(pubBytes, sigBytes); err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), message, sigBytes) {
		return ErrSignatureVerify
	}
	return nil
}
