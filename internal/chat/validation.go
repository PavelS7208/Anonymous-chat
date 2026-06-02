package chat

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func ValidateMessage(msg string) error {
	if len(msg) == 0 {
		return fmt.Errorf("empty message")
	}
	if len(msg) > MaxMessageBytes {
		return fmt.Errorf("message too long")
	}
	if !utf8.ValidString(msg) {
		return fmt.Errorf("invalid utf8")
	}
	if strings.ContainsAny(msg, "\n\r") {
		return fmt.Errorf("contains newline")
	}
	if unicode.IsSpace(rune(msg[0])) || unicode.IsSpace(rune(msg[len(msg)-1])) {
		return fmt.Errorf("starts or ends with whitespace")
	}
	return nil
}
