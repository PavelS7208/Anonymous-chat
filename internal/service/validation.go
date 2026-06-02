package service

import (
	"anonymous-chat/internal/chat"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func ValidateMessage(msg string) error {
	if len(msg) == 0 {
		return fmt.Errorf("empty message: %w", chat.ErrInvalidMessage)
	}
	if len(msg) > cfg.maxMessageBytes {
		return fmt.Errorf("message too long, expected %d bytes: %w", cfg.maxMessageBytes, chat.ErrInvalidMessage)
	}
	if !utf8.ValidString(msg) {
		return fmt.Errorf("invalid utf8 format: %w", chat.ErrInvalidMessage)
	}
	if strings.ContainsAny(msg, "\n\r") {
		return fmt.Errorf("contains newline: %w", chat.ErrInvalidMessage)
	}

	// Безопасная проверка первого и последнего символа (взять просто 0 и len-1 нельзя)
	first, _ := utf8.DecodeRuneInString(msg)
	last, _ := utf8.DecodeLastRuneInString(msg)
	if unicode.IsSpace(first) || unicode.IsSpace(last) {
		return fmt.Errorf("starts or ends with whitespace: %w", chat.ErrInvalidMessage)
	}
	return nil
}

func ValidateRoomName(roomName string) error {
	if !cfg.roomNameRe.MatchString(roomName) {
		return chat.ErrInvalidRoomName
	}
	return nil
}
