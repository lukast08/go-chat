package server

import (
	"context"
	"encoding/json"
	"log/slog"

	"server/internals/tcpserver"
)

type Sender struct {
	messageReader MessageReader
	publisher     Publisher
	logger        *slog.Logger
}

type Publisher interface {
	Publish(ctx context.Context, msg []byte) error
}

type MessageReader interface {
	ReceiveFromAll(ctx context.Context) <-chan tcpserver.Message
}

func NewSender(mr MessageReader, p Publisher, logger *slog.Logger) *Sender {
	return &Sender{
		messageReader: mr,
		publisher:     p,
		logger:        logger,
	}
}

func (s *Sender) SendMessagesToQueue(ctx context.Context) {
	msgChan := s.messageReader.ReceiveFromAll(ctx)
	for {
		select {
		case msg := <-msgChan:
			s.logger.Info(
				"sending message to queue",
				slog.String("clientID", msg.SenderID),
				slog.Int("nBytes", len(msg.Body)),
			)

			b, err := json.Marshal(msg)
			if err != nil {
				s.logger.Error("failed to marshal message", slog.Any("err", err))
			}

			err = s.publisher.Publish(ctx, b)
			if err != nil {
				s.logger.Error("failed to publish message", slog.Any("err", err))
			}
		case <-ctx.Done():
			return
		}
	}
}
