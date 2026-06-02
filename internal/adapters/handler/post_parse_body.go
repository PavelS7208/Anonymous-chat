package handler

import (
	"bytes"
	"errors"
)

// ParsePostBody парсит формат: {pubkey} {sig} {message}\n
// По спецификации протокола:
//   - {pubkey} и {sig} — base64-строки, разделённые одним или более пробелами
//   - {message} — всё от первого символа после {sig} (и любых пробелов-разделителей)
//     до завершающего \n (не включительно)
//   - Завершающий \n ОБЯЗАТЕЛЕН; его отсутствие — ошибка протокола (400)
//   - Возвращаемый content НЕ включает завершающий \n (это транспортный терминатор)
//   - Клиент должен подписывать ровно тот content, который возвращает эта функция
//     (без \n, но с любыми другими байтами внутри)
//
// Lenient-парсинг: допускает множественные пробелы между полями.
// Пробелы внутри message сохраняются как есть.
func ParsePostBody(b []byte) (pubKey, sig string, content []byte, err error) {
	// Протокольная валидация: сообщение должно заканчиваться на \n
	if len(b) == 0 || b[len(b)-1] != '\n' {
		return "", "", nil, errors.New("missing protocol terminator: message must end with \\n")
	}
	body := b[:len(b)-1]
	// убираем \r если есть (для win клиентов)
	if len(body) > 0 && body[len(body)-1] == '\r' {
		body = body[:len(body)-1]
	}
	// Найти конец pubkey (первый пробел) убирая ведущие пробелы впереди
	body = bytes.TrimLeft(body, " ")
	idx1 := bytes.IndexByte(body, ' ')
	if idx1 == -1 {
		return "", "", nil, errors.New("missing pubkey delimiter")
	}
	pubKey = string(body[:idx1])

	// Пропустить все пробелы после pubkey
	rest := body[idx1+1:]
	start := 0
	for start < len(rest) && rest[start] == ' ' {
		start++
	}
	if start >= len(rest) {
		return "", "", nil, errors.New("missing signature")
	}

	// Найти конец sig (следующий пробел после пропуска)
	idx2 := bytes.IndexByte(rest[start:], ' ')
	if idx2 == -1 {
		return "", "", nil, errors.New("missing signature delimiter")
	}
	sig = string(rest[start : start+idx2])

	// Пропустить пробелы после sig, остальное — сообщение + терминатор
	contentStart := start + idx2 + 1
	for contentStart < len(rest) && rest[contentStart] == ' ' {
		contentStart++
	}
	content = rest[contentStart:]

	return pubKey, sig, content, nil
}
