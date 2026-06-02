package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
)

// Ed25519Provider реализует крипто-интерфейс через стандартную библиотеку.
type Ed25519Provider struct{}

// NewEd25519Provider создает новый провайдер.
func NewEd25519Provider() *Ed25519Provider {
	return &Ed25519Provider{}
}

// GenerateKeyPair генерирует пару ключей согласно RFC 8032.
// Возвращает приватный seed (первые 32 байта) и публичный ключ (следующие 32).
func (p *Ed25519Provider) GenerateKeyPair() (privateSeed, pubKey []byte, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("ed25519.GenerateKey: %w", err)
	}
	// В Go crypto/ed25519:
	// - priv — 64 байта: первые 32 = seed (RFC 8032), следующие 32 = derived public
	// - pub — 32 байта
	privateSeed = make([]byte, 32)
	copy(privateSeed, priv[:32])
	pubKey = make([]byte, 32)
	copy(pubKey, pub)
	return privateSeed, pubKey, nil
}

// Verify проверяет подпись Ed25519.
func (p *Ed25519Provider) Verify(pubKey, message, signature []byte) bool {
	// ed25519.Verify ожидает:
	// - pubKey: 32 байта
	// - message: любые байты
	// - signature: 64 байта
	// Возвращает bool
	return ed25519.Verify(pubKey, message, signature)
}
