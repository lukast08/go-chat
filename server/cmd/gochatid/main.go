package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"server/cmd/gochatid/server"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	s, err := server.NewTCPServer(ctx)
	if err != nil {
		panic(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error)
	s.Start(ctx, errChan)

	select {
	// detect termination from console to shut down launched goroutines
	case <-sigs:
		log.Println("terminating server...")
	case err := <-errChan:
		log.Println("server encountered an error:", err)
	}
	cancel()
	<-time.After(time.Second * 3) // graceful shutdown
}
