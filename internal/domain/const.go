package domain

import (
	"fmt"
	"regexp"
)

type historyStrategyType int

const (
	historyStrategySlice historyStrategyType = iota
	historyStrategyRing
)

const (
	// Реализация хранения истории в памяти:
	// просто массив с обрезанием лимита
	// или круговой буфер без аллокаций, но постоянного размера равного лимит
	historyStrategy = historyStrategySlice
	//historyStrategy = historyStrategyRing

	// Участники диапазон нумерации
	minMemberID MemberID = 1
	maxMemberID MemberID = 1<<31 - 1

	// Длина сообщения максимальная для отправки
	maxMessageBytes = 1024
)

// Шаблон разрешенных символов и форматов имени комнаты
var roomNamePattern = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

func init() {
	// Проверяем согласованность констант при старте пакета
	// Паникует ДО main() — невозможно запустить с неверными инвариантами
	// Минимальная проверка что что-то озознаное проходит для имени комнаты
	if err := ValidateRoomName("test-room"); err != nil {
		panic(fmt.Sprintf("domain invariant broken: test-room must be valid: %v", err))
	}
	if minMemberID >= maxMemberID {
		panic(fmt.Sprintf("domain invariant broken: minMemberID(%d) >= maxMemberID(%d)",
			minMemberID, maxMemberID))
	}
}
