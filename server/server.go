package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	serverHost              = "localhost"
	serverPort              = "8080"
	letterBytes             = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	maxConnections          = 3
	connectionIDLength      = 10
	messagesChanSize        = 10
	clientMessageBufferSize = 1024
)

type connectionID string

type TCPServer struct {
	listener        net.Listener
	connectionsLock sync.Mutex
	connections     map[connectionID]net.Conn
}

func NewTCPServer(ctx context.Context) *TCPServer {
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", serverHost+":"+serverPort)
	if err != nil {
		panic(err)
	}
	return &TCPServer{
		listener:    listener,
		connections: make(map[connectionID]net.Conn),
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	server := NewTCPServer(ctx)
	defer server.listener.Close()

	log.Println("Server Running...")
	log.Println("Listening on " + serverHost + ":" + serverPort)
	log.Println("Waiting for client...")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func(ctx context.Context) {
		messagesChan := make(chan Message, messagesChanSize)
		go server.sendMessagesToAllConnections(ctx, messagesChan)

		for {
			connection, err := server.listener.Accept()
			if err != nil {
				fmt.Println("Error accepting: ", err.Error())
				os.Exit(1)
			}

			go func(ctx context.Context) {
				client, err := server.acceptClient(connection)
				if err != nil {
					log.Printf("failed to accept a client: %s\n", err.Error())
					return
				}

				go server.processClient(ctx, client, messagesChan)
			}(ctx)

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}(ctx)

	// wait for termination from console
	<-sigs
	cancel()
	log.Println("terminating server...")
	<-time.After(time.Second * 3)
}

type Message struct {
	senderID connectionID
	message  string
}

func (s *TCPServer) sendMessagesToAllConnections(ctx context.Context, messages chan Message) {
	select {
	case <-ctx.Done():
		return
	default:
		for message := range messages {
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

type Client struct {
	ID         connectionID
	Connection net.Conn
}

// TODO implement client termination after no message was read in some given period
func (s *TCPServer) acceptClient(connection net.Conn) (*Client, error) {
	if len(s.connections) == maxConnections {
		_, _ = connection.Write([]byte("server has reached maximum connection\n"))
		_ = connection.Close()
		return nil, fmt.Errorf("max capacity reached")
	}

	id := s.generateConnectionID()

	s.connectionsLock.Lock()
	s.connections[id] = connection
	s.connectionsLock.Unlock()

	log.Printf("client with ID: %s connected\n", id)

	_, _ = connection.Write([]byte("Welcome to the chat room!\n"))

	return &Client{
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

func (s *TCPServer) disconnectClient(client *Client) {
	_ = client.Connection.Close()
	delete(s.connections, client.ID)
	log.Printf("client with ID: %s disconnected\n", client.ID)
}

func (s *TCPServer) processClient(ctx context.Context, client *Client, shareChan chan Message) {
	defer s.disconnectClient(client)

	go func(ctx context.Context, mc chan Message) {
		for {
			buffer := make([]byte, clientMessageBufferSize)
			mLen, err := client.Connection.Read(buffer)
			if err != nil {
				return
			}

			mc <- Message{
				senderID: client.ID,
				message:  string(buffer[:mLen]),
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}(ctx, shareChan)

	select {
	case <-ctx.Done():
		return
	}
}
