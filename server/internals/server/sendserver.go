package server

import (
	"context"
	"encoding/json"
	"log/slog"

	"server/internals/log"
)

type Sender struct {
	messageReader MessageReader
	publisher     Publisher
}

type Publisher interface {
	Publish(ctx context.Context, msg []byte) error
}

type MessageReader interface {
	ReceiveFromAll(ctx context.Context) <-chan []byte
}

func NewSender(mr MessageReader, p Publisher) *Sender {
	return &Sender{
		messageReader: mr,
		publisher:     p,
	}
}

func (s *Sender) SendMessagesToQueue(ctx context.Context) {
	go func(ctx context.Context) {
		msgChan := s.messageReader.ReceiveFromAll(ctx)
		for {
			for msg := range msgChan {
				b, err := json.Marshal(msg)
				if err != nil {
					slog.Error("failed to marshal message", log.ErrorAttr(err))
				}

				err = s.publisher.Publish(ctx, b)
				if err != nil {
					slog.Error("failed to publish message", log.ErrorAttr(err))
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}(ctx)
}
