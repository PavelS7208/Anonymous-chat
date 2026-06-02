package service

type JoinRequest struct {
	RoomName string
	IP       string
}

type SendRequest struct {
	RoomName  string
	IP        string
	PubKeyB64 string
	SigB64    string
	Message   []byte
}
