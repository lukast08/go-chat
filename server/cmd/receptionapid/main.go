package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"

	"server/cmd/receptionapid/server"
	"server/internals/log"
	"server/internals/rmqclient"
	"server/internals/tcpserver"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	conf := tcpserver.Config{}
	err := envconfig.Process("receptionapi", &conf)
	if err != nil {
		panic(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	s, err := tcpserver.NewTCPServer(ctx, conf)
	if err != nil {
		panic(err)
	}

	errChan := make(chan error)
	go s.StartAcceptingConnections(ctx, errChan)

	// TODO configurable
	mqClient, err := rmqclient.NewRMQClient("guest", "localhost:5672", "messages")
	if err != nil {
		panic(err)
	}

	sndr := server.NewSender(s, mqClient)
	go sndr.SendMessagesToQueue(ctx)

	slog.Info("reception-api serving...")

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
