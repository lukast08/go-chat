package tcpserver

import (
	"math/rand"
	"net"
)

const (
	letterBytes        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	connectionIDLength = 10
)

type client struct {
	ID         string
	connection net.Conn
}

func newClient(connection net.Conn) *client {
	return &client{
		ID:         generateConnectionID(),
		connection: connection,
	}
}

func generateConnectionID() string {
	b := make([]byte, connectionIDLength)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}

func (c *client) sendMessage(msg []byte) (int, error) {
	return c.connection.Write(msg)
}

func (c *client) receive(buf []byte) (int, error) {
	return c.connection.Read(buf)
}

func (c *client) close() error {
	return c.connection.Close()
}
