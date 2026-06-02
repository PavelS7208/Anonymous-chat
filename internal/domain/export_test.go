package domain

// domain/export_test.go  специальный файл только для тестов
// Компилируется ТОЛЬКО с тегом test — в прод не попадает

// RoomWithConfig создаёт комнату с кастомным конфигом.
// Только для тестов — в продакшене использовать RoomFactoryDefault.
func RoomWithConfig(name string, cfg RoomConfig) *Room {
	return &Room{
		cfg:  &cfg,
		name: name,
		// Добавим как будем реализовывать тесты
	}
}

// Только для тестов — в продакшене использовать MemberFactoryDefault.
func MemberWithConfig(name string, cfg MemberConfig) *Member {
	return &Member{

		// Добавим как будем реализовывать тесты
	}
}
