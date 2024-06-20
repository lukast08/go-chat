package tcpserver

type Message struct {
	SenderID string `json:"sender_id"`
	Body     []byte `json:"body"`
}
