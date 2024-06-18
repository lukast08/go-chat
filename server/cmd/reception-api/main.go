package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"

	"server/cmd/reception-api/server"
	"server/internals/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	conf := server.Config{}
	err := envconfig.Process("receptionapi", &conf)
	if err != nil {
		panic(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	s, err := server.NewTCPServer(ctx, conf)
	if err != nil {
		panic(err)
	}

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
