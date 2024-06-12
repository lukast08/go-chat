package log

import (
	"log/slog"
)

func ErrorAttr(err error) slog.Attr {
	return slog.Attr{
		Key:   "err",
		Value: slog.AnyValue(err),
	}
}
