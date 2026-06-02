package domain

type Bootstrap struct {
	id          MemberID
	privateSeed []byte
}

func NewBootstrap(id MemberID, privateSeed []byte) Bootstrap {
	return Bootstrap{id: id, privateSeed: privateSeed}
}
func (b Bootstrap) Marshal() []byte {
	return formatBootstrap(b.id, b.privateSeed)
}
