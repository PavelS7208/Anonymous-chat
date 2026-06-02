package chat

import (
	"sync"
	"testing"
)

// Тестим что Close не паникует при повторных вызовах
func TestMember_Close_Idempotent(t *testing.T) {
	m := NewMember(MemberID(1), "gfhdghdgfh")

	// Закрыть 10 раз параллельно — не должно быть паники
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Close()
		}()
	}
	wg.Wait()

	// Убедиться, что канал закрыт
	_, ok := <-m.Done()
	if ok {
		t.Error("Expected done channel to be closed")
	}
}
