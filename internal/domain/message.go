package domain

import (
	"encoding/base64"

	"github.com/pavel/anonymous-chat/internal/domain/crypto"
)

// Message — бизнес сущность входящего сообщения из POST-запроса.
// Вход: сырые данные после парсинга протокола: [pubkey], [signature}, [message/content].
//
// Жизненный цикл:
// 1. NewMessage() — парсинг, декодирование, валидация формата и бизнес-правил
// 2. .Verify() — явная криптографическая верификация подписи
// 3. .ToEvent() — конвертация в domain.Event для рассылки (только после Verify!)
//
// Неизменяем после создания. Потокобезопасен для чтения
type Message struct {
	// pubKeyB64 — base64-представление публичного ключа отправителя.
	pubKeyB64 string

	// sigRaw — сырая подпись (64 байта), декодированная из base64 один раз.
	// Используется в .Verify() для криптографической проверки.
	// Хранится как []byte для эффективности: не декодировать при каждой верификации.
	sigRaw []byte

	// content — валидированное тело сообщения (сырые байты).
	// Гарантии после успешного NewMessage():
	//   - len(content) ∈ [1, 1024]
	//   - валидный UTF-8
	//   - без \n, \r внутри
	//   - без пробелов и переносов по краям
	content []byte
}

// NewMessage парсит и валидирует входящие данные сообщения
func NewMessage(pubKeyB64, sigB64 string, content []byte) (Message, error) {

	// Валидация base64 полей
	if err := validatePublicKey(pubKeyB64); err != nil {
		return Message{}, err
	}

	sigRaw, err := validateAndDecodeSignature(sigB64)
	if err != nil {
		return Message{}, err
	}

	// Валидация контента по бизнес-правилам
	if err := validateContent(content); err != nil {
		return Message{}, err
	}

	return Message{
		pubKeyB64: pubKeyB64, // храним как есть для lookup
		sigRaw:    sigRaw,    // декодирован для crypto.Verify
		content:   content,   // валидированные сырые байты
	}, nil
}

// Verify выполняет криптографическую верификацию подписи сообщения.
// Параметр cp — реализация крипто-провайдера (инъекция зависимости).
//
// Возвращает:
//   - nil: если подпись валидна и соответствует сообщению и ключу
//   - error: если верификация не удалась (невалидная подпись, ключ и т.д.)
//
// Важно: это явный шаг. Нельзя отправить сообщение в эфир без вызова Verify().
func (m Message) Verify(cp crypto.Provider) error {
	pubKeyRaw, err := base64.StdEncoding.DecodeString(m.pubKeyB64)
	if err != nil || len(pubKeyRaw) != 32 {
		// Это не должно произойти, если конструктор отработал корректно, но проверяем для безопасности
		return ErrPubKeyInvalidLength
	}

	// Выполняем криптографическую верификацию
	// cp.Verify — "тяжёлая" операция (Ed25519)
	if !cp.Verify(pubKeyRaw, m.content, m.sigRaw) {
		return ErrSignatureVerificationFailed
	}
	return nil
}

// ToEvent конвертирует верифицированное сообщение в доменное событие.
// Параметр senderID — идентификатор участника, отправившего сообщение.
// Возвращает domain.Event с типом EventMsg и валидированным контентом.
// Важные замечания: вызывать ТОЛЬКО после успешного .Verify()!
func (m Message) ToEvent(senderID MemberID) Event {
	// Копия контента для безопасности: внешние пакеты не должны иметь возможность
	// изменить внутреннее состояние через ссылку на срез.
	contentCopy := make([]byte, len(m.content))
	copy(contentCopy, m.content)
	return NewMsgEvent(senderID, contentCopy)
}

// SenderPubKeyB64 возвращает base64-представление публичного ключа отправителя.
func (m Message) SenderPubKeyB64() string {
	return m.pubKeyB64
}

// Content возвращает сырые байты валидированного сообщения.
// Используется для:
//   - Криптографической верификации (уже декодировано, готово к использованию)
//   - Форматирования в протокол (zero-copy append)
//   - Логирования (через string(msg.Content()))
//
// Возвращает ссылку на внутренний срез. Контракт: не модифицировать возвращённые байты.
// Если нужна гарантия неизменяемости — используйте bytes.Clone() на стороне вызывающего.
// Потокобезопасен для чтения, может вызываться из любой горутины.
func (m Message) Content() []byte {
	return m.content
}
