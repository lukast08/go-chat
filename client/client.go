package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
)

// TODO rewrite client

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
				log.Fatalln("Error reading:", err.Error())
				return
			}

			slog.Info("reading from dispatcher service", slog.Int("nBytes", n))

			if n > 0 {
				fmt.Print(string(buff[:n]))
			}

			select {
			case <-ctx.Done():
				break
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
