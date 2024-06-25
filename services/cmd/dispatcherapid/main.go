package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"

	"server/cmd/dispatcherapid/server"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	conf := server.Conf{}
	err := envconfig.Process("dispatcherapi", &conf)
	if err != nil {
		panic(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	errChan := make(chan error)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	go server.Start(ctx, conf, logger, errChan)

	select {
	// detect termination from console to shut down launched goroutines
	case <-sigs:
		slog.Info("terminating server...")
	case err := <-errChan:
		slog.ErrorContext(ctx, "server encountered an error", slog.Any("err", err))
	}
	cancel()
	<-time.After(time.Second * 3) // graceful shutdown
}
