package tcpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"server/internals/log"
)

const (
	clientMessageBufferSize = 1024
)

type Config struct {
	Port           string `default:":8080"`
	MaxConnections int    `default:"3"`
}

type TCPServer struct {
	listener        net.Listener
	connectionsLock sync.RWMutex
	clients         map[string]*client
	newClientChan   chan string

	maxConnections int
}

func NewTCPServer(ctx context.Context, conf Config) (*TCPServer, error) {
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", conf.Port)
	if err != nil {
		return nil, err
	}

	s := &TCPServer{
		listener:       listener,
		clients:        make(map[string]*client),
		maxConnections: conf.MaxConnections,
		newClientChan:  make(chan string, conf.MaxConnections),
	}

	return s, nil
}

func (s *TCPServer) StartAcceptingConnections(ctx context.Context, errChan chan error) {
	slog.Info("Server Running...")
	slog.Info("Waiting for client...")

	for {
		connection, err := s.listener.Accept()
		if err != nil {
			errChan <- fmt.Errorf("error accepting connection: %w", err)
		}

		// running as a goroutine so that server can accept multiple clients at the same time
		go func(ctx context.Context, connection net.Conn) {
			err := s.acceptClient(connection)
			if err != nil {
				slog.Error("failed to accept a client:", log.ErrorAttr(err))
				return
			}
		}(ctx, connection)

		select {
		case <-ctx.Done():
			break
		default:
		}
	}
}

var errMaxCapacity = errors.New("maximum connection capacity reached")

func (s *TCPServer) acceptClient(connection net.Conn) error {
	s.connectionsLock.Lock()
	if len(s.clients) == s.maxConnections {
		s.connectionsLock.Unlock()
		_, err := connection.Write([]byte("Chat room has reached maximum clients\n"))
		if err != nil {
			return err
		}

		err = connection.Close()
		if err != nil {
			return err
		}

		return errMaxCapacity
	}

	c := newClient(connection)
	s.clients[c.ID] = c
	s.newClientChan <- c.ID
	s.connectionsLock.Unlock()

	slog.Info("client connected", slog.String("clientID", c.ID))

	return nil
}

func (s *TCPServer) ReceiveFromAll(ctx context.Context) <-chan Message {
	msgChan := make(chan Message, 1_000) // arbitrary buffer size for now, // TODO get back to

	go func(ctx context.Context, msgChan chan Message) {
		for {
			select {
			case connID := <-s.newClientChan:
				go s.processClient(ctx, s.clients[connID], msgChan)
			case <-ctx.Done():
				return
			}
		}
	}(ctx, msgChan)

	return msgChan
}

func (s *TCPServer) processClient(ctx context.Context, client *client, msgChan chan Message) {
	slog.Info("starting processing client", slog.String("clientID", client.ID))

	for {
		buffer := make([]byte, clientMessageBufferSize)
		mLen, err := client.receive(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.disconnectClient(client)
				return
			}

			slog.Error("failed to read from client:", log.ErrorAttr(err))
		}
		slog.Info("received message from client", slog.String("clientID", client.ID), slog.Int("nBytes", mLen))

		msgChan <- Message{Body: string(buffer[:mLen]), SenderID: client.ID}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (s *TCPServer) disconnectClient(client *client) {
	_ = client.close()

	s.connectionsLock.Lock()
	delete(s.clients, client.ID)
	s.connectionsLock.Unlock()

	slog.Info("client disconnected", slog.String("clientID", string(client.ID)))
}

type writeToAllErr struct {
	nConnections int
	errs         []error
}

func (err writeToAllErr) Error() string {
	return fmt.Sprintf(
		" %d/%d writes failed, first error: %s",
		len(err.errs),
		err.nConnections,
		err.errs[0],
	)
}

func (s *TCPServer) WriteToAll(msg []byte) error {
	// not pre-allocating, as most of the time, hopefully, there should be no errors and nil will be returned
	var errs []error
	s.connectionsLock.RLock()
	for _, client := range s.clients {
		_, err := client.sendMessage(msg)
		if err != nil {
			errs = append(errs, err)
		}
	}
	s.connectionsLock.RUnlock()

	if errs != nil {
		return writeToAllErr{
			nConnections: len(s.clients),
			errs:         errs,
		}
	}

	return nil
}
