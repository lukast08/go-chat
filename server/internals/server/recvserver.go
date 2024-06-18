package server

import (
	"context"
	"log/slog"

	"server/internals/log"
)

type ReceiverServer struct {
	messageWriter MessageWriter
	consumer      Consumer
}

type Consumer interface {
	StartConsuming(ctx context.Context) (<-chan []byte, error)
}

type MessageWriter interface {
	WriteToAll(msg []byte) error
}

func NewReceiver(mw MessageWriter, c Consumer) *ReceiverServer {
	return &ReceiverServer{
		messageWriter: mw,
		consumer:      c,
	}
}

func (rs *ReceiverServer) Start(ctx context.Context) {
	go func(ctx context.Context) {
		msgChan, err := rs.consumer.StartConsuming(ctx)
		if err != nil {
			panic(err) // if I can not start consuming, I don't want the service to start
		}

		for msg := range msgChan {
			err := rs.messageWriter.WriteToAll(msg)
			if err != nil {
				slog.Error("failed to send messages", log.ErrorAttr(err))
			}
		}
	}(ctx)
}
