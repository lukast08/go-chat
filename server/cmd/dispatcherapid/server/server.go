package server

import (
	"context"
	"log/slog"

	"server/internals/rmqclient"
	"server/internals/tcpserver"
)

type Conf struct {
	SocketPort     string `default:":8799"`
	MaxConnections int    `required:"true"`
	MQConnection   string `required:"true"`
	MQUser         string `default:"guest"`
	MQName         string `default:"messages"`
}

func Start(ctx context.Context, conf Conf, logger *slog.Logger, errChan chan error) {
	s, err := tcpserver.NewTCPServer(
		ctx,
		logger,
		tcpserver.Config{
			Port:           conf.SocketPort,
			MaxConnections: conf.MaxConnections,
		},
	)
	if err != nil {
		errChan <- err
	}

	go s.StartAcceptingConnections(ctx)

	mqClient, err := rmqclient.NewRMQClient(conf.MQUser, conf.MQConnection, conf.MQName)
	if err != nil {
		errChan <- err
	}

	sndr := NewReceiver(s, mqClient, logger)
	go sndr.StartReceiving(ctx)

	logger.Info("dispatcher-api serving...")
}
