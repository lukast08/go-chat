package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
)

// TODO rewrite client

type message struct {
	SenderID string `json:"sender_id"`
	Body     string `json:"body"`
}

func connectDispatcher(ctx context.Context) error {
	connection, err := net.Dial("tcp", "localhost:8799")
	if err != nil {
		return err
	}

	go func() {
		for {
			buff := make([]byte, 1024)
			n, err := connection.Read(buff)
			if err != nil {
				slog.Error("failed to read from the server", slog.Any("err", err))
				return
			}

			slog.Info("reading from dispatcher service", slog.Int("nBytes", n))
			if n > 0 {
				var msg message
				err := json.Unmarshal(buff[:n], &msg)
				if err != nil {
					slog.Error("failed to unmarshal incoming message", slog.Any("err", err))
				}

				fmt.Printf("(%s): %s", msg.SenderID, msg.Body)
			}

			select {
			case <-ctx.Done():
				break
			default:
			}
		}
	}()

	return nil
}

func connectReception(ctx context.Context, msgChan chan string) error {
	connection, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case msg := <-msgChan:
				n, err := connection.Write([]byte(msg))
				if err != nil {
					panic(err)
				}
				slog.Info("wrote to reception service", slog.Int("nBytes", n))
			case <-ctx.Done():
				break
			}
		}
	}()

	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := connectDispatcher(ctx)
	if err != nil {
		panic(err)
	}

	msgChan := make(chan string)
	err = connectReception(ctx, msgChan)
	if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		str, _ := reader.ReadString('\n')

		msgChan <- str
	}
}
