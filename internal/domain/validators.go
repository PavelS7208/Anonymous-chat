package domain

import (
	"bytes"
	"encoding/base64"
	"unicode"
	"unicode/utf8"
)

func ValidateRoomName(name string) error {
	if !roomNamePattern.MatchString(name) {
		return ErrInvalidRoomName
	}
	return nil
}

// validateContent проверяет тело сообщения согласно бизнес-правилам:
//   - Не пустое (длина >= 1)
//   - Длина <= domain.maxMessageBytes (по умолчанию 1024)
//   - Валидный UTF-8 (строгая проверка, без суррогатных пар)
//   - Не содержит символов переноса строки (\n, \r)
//   - Не начинается и не заканчивается пробельными символами (Unicode-aware)
//
// Возвращает nil, если все проверки пройдены, или конкретную ошибку домена.
// Используется внутри message.NewMessage() для валидации входящих данных.
func validateContent(content []byte) error {

	// Пустое сообщение — ошибка (бизнес-правило)
	if len(content) == 0 {
		return ErrMessageEmpty
	}

	if len(content) > maxMessageBytes {
		return ErrMessageTooLong
	}

	// 3. Валидный UTF-8 (строгая проверка)
	// utf8.Valid проверяет всю последовательность, включая завершённость символов
	if !utf8.Valid(content) {
		return ErrMessageInvalidUTF8
	}

	// 4. Запрет переносов строки (\n, \r) жестко и в конце тоже
	if bytes.ContainsAny(content, "\n\r") {
		return ErrMessageContainsNewline
	}

	// 5. Запрет пробельных символов по краям (Unicode-aware)
	// Используем utf8.DecodeRune для корректной работы с многобайтовыми символами
	firstRune, _ := utf8.DecodeRune(content)
	lastRune, _ := utf8.DecodeLastRune(content)
	if unicode.IsSpace(firstRune) || unicode.IsSpace(lastRune) {
		return ErrMessageWhitespaceAtEdge
	}

	return nil
}

func validatePublicKey(pubKeyB64 string) error {
	// Валидация формата pubKeyB64 (без декодирования для экономии)
	// Проверяем, что это валидный base64 и декодируется в 32 байта
	decodedPub, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil {
		return ErrInvalidPubKeyBase64
	}
	if len(decodedPub) != 32 {
		return ErrPubKeyInvalidLength
	}
	return nil
}

func validateAndDecodeSignature(sigB64 string) ([]byte, error) {

	// Декодирование и валидация подписи (один раз, для крипто-операций)
	sigRaw, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, ErrInvalidSignature
	}
	if len(sigRaw) != 64 {
		return nil, ErrSignatureInvalidLength
	}
	return sigRaw, nil
}
