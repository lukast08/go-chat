package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"sync"

	"server/internals/log"
)

const (
	letterBytes             = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	connectionIDLength      = 10
	messagesChanSize        = 10
	clientMessageBufferSize = 1024
)

type Config struct {
	Port           string `default:":8080"`
	MaxConnections int    `default:"3"`
}

type connectionID string

type TCPServer struct {
	listener        net.Listener
	connectionsLock sync.Mutex
	connections     map[connectionID]net.Conn
	messChan        chan message

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
		connections:    make(map[connectionID]net.Conn),
		messChan:       make(chan message, messagesChanSize),
		maxConnections: conf.MaxConnections,
	}

	go s.sendMessagesToAllConnections(ctx)

	return s, nil
}

func (s *TCPServer) Start(ctx context.Context, errChan chan error) {
	slog.Info("Server Running...")
	slog.Info("Waiting for client...")

	for {
		connection, err := s.listener.Accept()
		if err != nil {
			errChan <- fmt.Errorf("error accepting connection: %w", err)
		}

		go func(ctx context.Context, connection net.Conn) {
			client, err := s.acceptClient(connection)
			if err != nil {
				slog.Error("failed to accept a client:", log.ErrorAttr(err))
				return
			}

			go s.processClient(ctx, client)
		}(ctx, connection)

		select {
		case <-ctx.Done():
			break
		default:
		}
	}

}

type message struct {
	senderID connectionID
	message  string
}

func (s *TCPServer) sendMessagesToAllConnections(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
		for message := range s.messChan {
			for userID, conn := range s.connections {
				if userID == message.senderID {
					continue
				}

				if message.message[len(message.message)-1:] != "\n" {
					message.message = message.message + "\n"
				}

				_, _ = conn.Write([]byte(fmt.Sprintf("(%s): %s", message.senderID, message.message)))
			}
		}
	}
}

type client struct {
	ID         connectionID
	Connection net.Conn
}

var errMaxCapacity = errors.New("maximum connection capacity reached")

func (s *TCPServer) acceptClient(connection net.Conn) (*client, error) {
	id := s.generateConnectionID()

	s.connectionsLock.Lock()
	if len(s.connections) == s.maxConnections {
		s.connectionsLock.Unlock()
		_, err := connection.Write([]byte("Chat room has reached maximum connections\n"))
		if err != nil {
			return nil, err
		}

		err = connection.Close()
		if err != nil {
			return nil, err
		}

		return nil, errMaxCapacity
	}

	s.connections[id] = connection
	s.connectionsLock.Unlock()

	slog.Info("client connected", slog.String("clientID", string(id)))

	_, err := connection.Write([]byte("Welcome to the chat room!\n"))
	if err != nil {
		return nil, err
	}

	return &client{
		ID:         id,
		Connection: connection,
	}, nil
}

func (s *TCPServer) generateConnectionID() connectionID {
	b := make([]byte, connectionIDLength)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return connectionID(b)
}

func (s *TCPServer) disconnectClient(client *client) {
	_ = client.Connection.Close()

	s.connectionsLock.Lock()
	delete(s.connections, client.ID)
	s.connectionsLock.Unlock()

	slog.Info("client with disconnected", slog.String("clientID", string(client.ID)))
}

func (s *TCPServer) processClient(ctx context.Context, client *client) {
	go func(ctx context.Context, mc chan message) {
		for {
			buffer := make([]byte, clientMessageBufferSize)
			mLen, err := client.Connection.Read(buffer)
			if err != nil {
				if errors.Is(err, io.EOF) {
					s.disconnectClient(client)
					return
				}

				slog.Error("failed to read from connection:", log.ErrorAttr(err))
			}

			mc <- message{
				senderID: client.ID,
				message:  string(buffer[:mLen]),
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}(ctx, s.messChan)

	select {
	case <-ctx.Done():
		return
	}
}
