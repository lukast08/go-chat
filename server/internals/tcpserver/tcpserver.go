package tcpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
)

const (
	clientMessageBufferSize = 1024
)

type Config struct {
	Port           string
	MaxConnections int
}

type TCPServer struct {
	listener        net.Listener
	connectionsLock sync.RWMutex
	clients         map[string]*client
	newClientChan   chan string
	logger          *slog.Logger

	maxConnections int
}

func NewTCPServer(ctx context.Context, logger *slog.Logger, conf Config) (*TCPServer, error) {
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", conf.Port)
	if err != nil {
		return nil, err
	}

	s := TCPServer{
		listener:       listener,
		clients:        make(map[string]*client),
		maxConnections: conf.MaxConnections,
		newClientChan:  make(chan string, conf.MaxConnections),
		logger:         logger,
	}

	return &s, nil
}

func (s *TCPServer) StartAcceptingConnections(ctx context.Context) {
	s.logger.Info("Server Running...")
	s.logger.Info("Waiting for client...")

	for {
		connection, err := s.listener.Accept()
		if err != nil {
			s.logger.Error("error accepting connection", ErrAttr(err))
		}

		// running as a goroutine so that server can accept multiple clients at the same time
		go func(ctx context.Context, connection net.Conn) {
			err := s.acceptClient(connection)
			if err != nil {
				s.logger.Error("failed to accept a client:", ErrAttr(err))
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

const maxCapacityMessage = "Chat room has reached maximum clients\n"

func (s *TCPServer) acceptClient(connection net.Conn) error {
	s.connectionsLock.Lock()
	if len(s.clients) == s.maxConnections {
		s.connectionsLock.Unlock()
		_, err := connection.Write([]byte(maxCapacityMessage))
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

	s.logger.Info("client connected", slog.String("clientID", c.ID))

	return nil
}

func (s *TCPServer) ReceiveFromAll(ctx context.Context) <-chan Message {
	// As per Uber go style: Channels should usually have a size of one or be unbuffered
	msgChan := make(chan Message, 1)

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
	s.logger.Info("starting processing client", slog.String("clientID", client.ID))

	buffer := make([]byte, clientMessageBufferSize)
	for {
		mLen, err := client.receive(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.disconnectClient(client)
				return
			}

			s.logger.Error("failed to read from client:", ErrAttr(err))
		}
		s.logger.Info("received message from client", slog.String("clientID", client.ID), slog.Int("nBytes", mLen))

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

	s.logger.Info("client disconnected", slog.String("clientID", string(client.ID)))
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
