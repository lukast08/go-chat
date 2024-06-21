package tcpserver

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTCPServer(t *testing.T) {
	conf := Config{
		Port:           ":0",
		MaxConnections: 3,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := NewTCPServer(ctx, conf)
	assert.NoError(t, err)
}

func TestAcceptClient(t *testing.T) {
	conf := Config{
		Port:           ":0",
		MaxConnections: 3,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := NewTCPServer(ctx, conf)
	assert.NoError(t, err)

	go server.StartAcceptingConnections(ctx, make(chan error))

	_, err = net.Dial("tcp", server.listener.Addr().String())
	assert.NoError(t, err)

	time.Sleep(1 * time.Second) // Wait for the server to process the connection

	server.connectionsLock.RLock()
	assert.Equal(t, 1, len(server.clients))
	server.connectionsLock.RUnlock()

	conns := make([]net.Conn, conf.MaxConnections)
	for i := 0; i < conf.MaxConnections; i++ {
		c, err := net.Dial("tcp", server.listener.Addr().String())
		assert.NoError(t, err)
		conns[i] = c
	}

	time.Sleep(1 * time.Second) // Wait for the server to process connections

	server.connectionsLock.RLock()
	assert.Equal(t, conf.MaxConnections, len(server.clients))
	server.connectionsLock.RUnlock()

	// one more connection should receive a message about max clients
	extraConn, err := net.Dial("tcp", server.listener.Addr().String())
	assert.NoError(t, err)

	buf := make([]byte, 1024)
	n, err := extraConn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, maxCapacityMessage, string(buf[:n]))
}

func TestReceiveFromAll(t *testing.T) {
	conf := Config{
		Port:           ":0",
		MaxConnections: 3,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := NewTCPServer(ctx, conf)
	assert.NoError(t, err)

	go server.StartAcceptingConnections(ctx, make(chan error))

	conn, err := net.Dial("tcp", server.listener.Addr().String())
	assert.NoError(t, err)

	time.Sleep(1 * time.Second) // Wait for the server to process the connection

	msgChan := server.ReceiveFromAll(ctx)
	expectedMsg := "Hello, World!"
	_, err = conn.Write([]byte(expectedMsg))
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		select {
		case msg := <-msgChan:
			return expectedMsg == msg.Body
		default:
			return false
		}
	}, time.Second, time.Millisecond*10)
}

func TestWriteToAll(t *testing.T) {
	conf := Config{
		Port:           ":0",
		MaxConnections: 3,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := NewTCPServer(ctx, conf)
	assert.NoError(t, err)

	go server.StartAcceptingConnections(ctx, make(chan error))

	conn1, err := net.Dial("tcp", server.listener.Addr().String())
	assert.NoError(t, err)

	conn2, err := net.Dial("tcp", server.listener.Addr().String())
	assert.NoError(t, err)

	time.Sleep(1 * time.Second) // Wait for the server to process connections

	msg := "message to all"
	err = server.WriteToAll([]byte(msg))
	assert.NoError(t, err)

	buf1 := make([]byte, len(msg))
	_, err = conn1.Read(buf1)
	assert.NoError(t, err)
	assert.Equal(t, msg, string(buf1))

	buf2 := make([]byte, len(msg))
	_, err = conn2.Read(buf2)
	assert.NoError(t, err)
	assert.Equal(t, msg, string(buf2))
}
