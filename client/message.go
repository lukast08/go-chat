package main

type message struct {
	SenderID string `json:"sender_id"`
	Body     []byte `json:"body"`
}
