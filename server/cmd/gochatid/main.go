package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"server/cmd/gochatid/server"
	"server/internals/log"
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
	go s.Start(ctx, errChan)

	select {
	// detect termination from console to shut down launched goroutines
	case <-sigs:
		slog.Info("terminating server...")
	case err := <-errChan:
		slog.ErrorContext(ctx, "server encountered an error", log.ErrorAttr(err))
	}
	cancel()
	<-time.After(time.Second * 3) // graceful shutdown
}
