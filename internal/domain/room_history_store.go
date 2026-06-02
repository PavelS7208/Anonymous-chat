package domain

// ----- Обычный slice в качестве хранилища ---------------
type sliceHistory struct {
	data         []Event
	limit        int
	snapshotSize int
}

func newSliceHistory(limit, snapshotSize int) *sliceHistory {
	return &sliceHistory{data: make([]Event, 0, min(limit, 64)), limit: limit, snapshotSize: snapshotSize}
}

func (h *sliceHistory) Push(evt Event) {
	h.data = append(h.data, evt)
	if len(h.data) > h.limit {
		oldStart := len(h.data) - h.limit
		for i := 0; i < oldStart; i++ {
			h.data[i].Message = nil
		}
		h.data = h.data[oldStart:]
	}
}

func (h *sliceHistory) Snapshot() []Event {
	total := len(h.data)
	if h.snapshotSize >= total {
		snap := make([]Event, total)
		copy(snap, h.data)
		return snap
	}
	snap := make([]Event, h.snapshotSize)
	copy(snap, h.data[total-h.snapshotSize:])
	return snap
}
func (h *sliceHistory) Len() int { return len(h.data) }

// ----- через кольцевой буфер без аллокаций -------

type ringHistory struct {
	buf          *eventRingBuf
	snapshotSize int
}

func newRingHistory(capacity, snapshotSize int) *ringHistory {
	h := &ringHistory{buf: newEventRingBuf(capacity), snapshotSize: snapshotSize}

	return h
}

func (h *ringHistory) Push(evt Event)    { h.buf.pushOverwrite(evt) }
func (h *ringHistory) Snapshot() []Event { return h.buf.lastN(h.snapshotSize) }
func (h *ringHistory) Len() int          { return h.buf.len() }
