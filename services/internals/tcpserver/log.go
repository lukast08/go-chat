package tcpserver

import (
	"context"
	"log/slog"
)

func ErrAttr(err error) slog.Attr {
	return slog.Attr{Key: "err", Value: slog.AnyValue(err)}
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (h discardHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}
func (h discardHandler) WithGroup(_ string) slog.Handler {
	return h
}
