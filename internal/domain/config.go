package domain

type RoomConfig struct {
	// История размер максимальный
	HistorySize int
	// Сколько отдаем новому участнику событий из истории. Дефолт 10 (по ТЗ)
	SnapshotSize int

	// Размер буфера комнаты для отправки сообщений участникам. Дефолт 256
	BroadcastQueueSize int
}

func (c RoomConfig) withDefaults() RoomConfig {
	if c.HistorySize == 0 {
		c.HistorySize = 100
	}
	if c.SnapshotSize == 0 {
		c.SnapshotSize = 50
	}
	if c.BroadcastQueueSize == 0 {
		c.BroadcastQueueSize = 256
	}
	return c
}

type MemberConfig struct {

	// EventChannelBufSize — размер буфера канала событий на участника.
	// Возможность задержки приема сообщений. Дефолт 64
	EventChannelBufSize int

	OverflowBufSize int // Буфер "последнего шанса" (дефолт 16)

}

func (c MemberConfig) withDefaults() MemberConfig {
	if c.OverflowBufSize == 0 {
		c.OverflowBufSize = 16
	}
	if c.EventChannelBufSize == 0 {
		c.EventChannelBufSize = 64
	}
	return c
}
