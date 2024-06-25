package server

import (
	"context"
	"log/slog"
)

type ReceiverServer struct {
	messageWriter MessageWriter
	consumer      Consumer
	logger        *slog.Logger
}

type Consumer interface {
	StartConsuming(ctx context.Context) (<-chan []byte, error)
}

type MessageWriter interface {
	WriteToAll(msg []byte) error
}

func NewReceiver(mw MessageWriter, c Consumer, logger *slog.Logger) *ReceiverServer {
	return &ReceiverServer{
		messageWriter: mw,
		consumer:      c,
		logger:        logger,
	}
}

// TODO bug with disconnected clients not deleted.

func (rs *ReceiverServer) StartReceiving(ctx context.Context) {
	msgChan, err := rs.consumer.StartConsuming(ctx)
	if err != nil {
		panic(err) // if I can not start consuming, I don't want the service to start
	}

	for {
		select {
		case msg := <-msgChan:
			rs.logger.Info("sending message to all clients")

			err = rs.messageWriter.WriteToAll(msg)
			if err != nil {
				rs.logger.Error("failed to send messages", slog.Any("err", err))
			}
		case <-ctx.Done():
			return
		}
	}
}
