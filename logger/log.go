package logger

import (
	"log/slog"
	"os"
)

type Logger interface {
	Info(msg string)
	Error(msg string, err error)
	Debug(msg string)
}

type DoodleLogger struct {
	logger *slog.Logger
}

func New(loggerName string) Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	attrs := []slog.Attr{slog.String("logger", loggerName)}
	h := handler.WithAttrs(attrs)
	logger := slog.New(h)
	return DoodleLogger{logger}
}

func (dl DoodleLogger) Info(msg string) {
	dl.logger.Info(msg)
}

func (dl DoodleLogger) Error(msg string, err error) {
	if err != nil {
		e := slog.String("error", err.Error())
		dl.logger.Error(msg, e)
		return
	}
	dl.logger.Error(msg)
}

func (dl DoodleLogger) Debug(msg string) {
	dl.logger.Debug(msg)
}
