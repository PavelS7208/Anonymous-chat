package domain

// eventRingBuf — внутреннее ядро кольцевого буфера для Event.
// Обеспечивает zero-allocation, O(1) операции и GC-safe очистку
type eventRingBuf struct {
	data  []Event
	head  int
	count int
	cap   int
}

func newEventRingBuf(capacity int) *eventRingBuf {
	return &eventRingBuf{data: make([]Event, capacity), cap: capacity}
}

func (b *eventRingBuf) len() int { return b.count }

func (b *eventRingBuf) pushOverwrite(evt Event) {
	b.data[b.head] = evt
	b.head = (b.head + 1) % b.cap
	if b.count < b.cap {
		b.count++
	}
}

func (b *eventRingBuf) pushReject(evt Event) bool {
	if b.count >= b.cap {
		return false
	}
	idx := (b.head + b.count) % b.cap
	b.data[idx] = evt
	b.count++
	return true
}

func (b *eventRingBuf) peek() (Event, bool) {
	if b.count == 0 {
		return Event{}, false
	}
	return b.data[b.head], true
}

func (b *eventRingBuf) pop() (Event, bool) {
	if b.count == 0 {
		return Event{}, false
	}
	evt := b.data[b.head]
	b.data[b.head] = Event{} // 🔑 Zero-out для GC
	b.head = (b.head + 1) % b.cap
	b.count--
	return evt, true
}

func (b *eventRingBuf) lastN(n int) []Event {
	if n <= 0 || b.count == 0 {
		return []Event{}
	}
	if n > b.count {
		n = b.count
	}
	start := (b.head - n + b.cap) % b.cap
	res := make([]Event, n)
	for i := 0; i < n; i++ {
		res[i] = b.data[(start+i)%b.cap]
	}
	return res
}
