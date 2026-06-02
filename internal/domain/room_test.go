package domain

import (
	"strconv"
	"testing"
)

// Стресс тест на последовательность seq в истории комнаты
func TestBroadcast_SeqOrder(t *testing.T) {
	r := NewRoom("test")
	const N = 1000
	done := make(chan bool, N)

	// Запускаем много горутин, которые шлют события
	for i := 0; i < N; i++ {
		go func(val int) {
			r.Broadcast(Event{Kind: EventMsg, Message: []byte(strconv.Itoa(val))})
			done <- true
		}(i)
	}

	// Ждём завершения
	for i := 0; i < N; i++ {
		<-done
	}

	// Проверяем историю: она должна быть отсортирована по Seq
	snapshot := r.history.LastN(N)
	for i := 1; i < len(snapshot); i++ {
		if snapshot[i].Seq <= snapshot[i-1].Seq {
			t.Errorf("History order violated: seq[%d]=%d <= seq[%d]=%d",
				i, snapshot[i].Seq, i-1, snapshot[i-1].Seq)
		}
	}
}
