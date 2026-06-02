package domain

type MemberFactory interface {
	NewMember(id MemberID, pubKeyB64 string) *Member
}

type MemberFactoryDefault struct {
	cfg MemberConfig
}

func NewMemberFactory(cfg MemberConfig) *MemberFactoryDefault {
	cfg = cfg.withDefaults()
	return &MemberFactoryDefault{cfg}
}

func (f *MemberFactoryDefault) NewMember(id MemberID, pubKeyB64 string) *Member {
	return &Member{
		id:        id,
		pubKeyB64: pubKeyB64,
		events:    make(chan Event, f.cfg.EventChannelBufSize),
		overflow:  newOverflowBuf(f.cfg.OverflowBufSize),
		done:      make(chan struct{}),
	}
}
