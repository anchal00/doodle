package logger

import (
	"log/slog"
	"os"
)

func NewLogger(loggerName string) *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})

	attrs := []slog.Attr{slog.String("logger", loggerName)}
	h := handler.WithAttrs(attrs)
	logger := slog.New(h)
	return logger
}
