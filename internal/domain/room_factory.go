package domain

type RoomFactory interface {
	NewRoom(name string) (*Room, error)
}

type RoomFactoryDefault struct {
	cfg RoomConfig
	mf  MemberFactory
}

func NewRoomFactory(cfg RoomConfig, mf MemberFactory) *RoomFactoryDefault {
	cfg = cfg.withDefaults()
	return &RoomFactoryDefault{cfg: cfg, mf: mf}
}

func (f *RoomFactoryDefault) NewRoom(name string) (*Room, error) {
	if err := ValidateRoomName(name); err != nil { // доменная валидация
		return nil, err
	}

	r := &Room{
		name:           name,
		memberFactory:  f.mf,
		members:        make(map[MemberID]*Member),
		memberByPubKey: make(map[string]*Member),
		history:        newHistory(historyStrategy, f.cfg.HistorySize, f.cfg.SnapshotSize),
		broadcast:      make(chan Event, f.cfg.BroadcastQueueSize),
	}
	r.nextMemberID.Store(int64(minMemberID) - 1)
	r.closed.Store(false)
	go r.Run()
	return r, nil
}

func newHistory(strategy historyStrategyType, cap, snapshotSize int) eventHistory {
	var hRepo eventHistory

	switch strategy {
	case historyStrategySlice:
		hRepo = newSliceHistory(cap, snapshotSize)
	default:
		hRepo = newRingHistory(cap, snapshotSize)
	}
	return hRepo
}
