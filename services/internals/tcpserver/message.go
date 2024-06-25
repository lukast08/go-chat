package tcpserver

type Message struct {
	SenderID string `json:"sender_id"`
	Body     string `json:"body"`
}
