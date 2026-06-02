package domain

// Адаптер Буфер (на основе кольцевого буфера событий) для "последнего шанса" отправит данные медленному клиенту

type memberOverflowBuf struct{ buf *eventRingBuf }

func newOverflowBuf(capacity int) *memberOverflowBuf {
	return &memberOverflowBuf{buf: newEventRingBuf(capacity)}
}
func (b *memberOverflowBuf) push(evt Event) bool { return b.buf.pushReject(evt) }
func (b *memberOverflowBuf) pop() (Event, bool)  { return b.buf.pop() }
func (b *memberOverflowBuf) len() int            { return b.buf.len() }
func (b *memberOverflowBuf) peek() (Event, bool) { return b.buf.peek() }
