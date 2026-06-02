package chat

const (
	InitialHistoryCount = 10                      // Длина списка сообщений для нового участника
	MaxMemberID         = (1 << 63) - 1           // Лимит индекса memberID
	MaxHistoryStorage   = 50000                   // Защита от OOM (обрезаем хвост истории)
	MaxMessageBytes     = 1024                    // Лимит длины сообщения
	EventChannelBuf     = 64                      // Буфер канала событий, выбрали кратно 2 (без всяких тестов)
	RoomNamePattern     = `^[a-zA-Z0-9_-]{1,64}$` // Регулярка для проверки на разрешенные символы в имени комнаты и длину
)
