package server

import (
	"context"
	"encoding/json"
	"log/slog"

	"server/internals/log"
	"server/internals/tcpserver"
)

type Sender struct {
	messageReader MessageReader
	publisher     Publisher
}

type Publisher interface {
	Publish(ctx context.Context, msg []byte) error
}

type MessageReader interface {
	ReceiveFromAll(ctx context.Context) <-chan tcpserver.Message
}

func NewSender(mr MessageReader, p Publisher) *Sender {
	return &Sender{
		messageReader: mr,
		publisher:     p,
	}
}

func (s *Sender) SendMessagesToQueue(ctx context.Context) {
	msgChan := s.messageReader.ReceiveFromAll(ctx)
	for {
		select {
		case msg := <-msgChan:
			slog.Info(
				"sending message to queue",
				slog.String("clientID", msg.SenderID),
				slog.Int("nBytes", len(msg.Body)),
			)

			b, err := json.Marshal(msg)
			if err != nil {
				slog.Error("failed to marshal message", log.ErrorAttr(err))
			}

			err = s.publisher.Publish(ctx, b)
			if err != nil {
				slog.Error("failed to publish message", log.ErrorAttr(err))
			}
		case <-ctx.Done():
			return
		}
	}
}
