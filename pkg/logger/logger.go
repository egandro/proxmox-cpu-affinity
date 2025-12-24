package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// SimpleHandler implements slog.Handler for common log format.
type SimpleHandler struct {
	Output io.Writer
	Level  slog.Level
}

func (h *SimpleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.Level
}

func (h *SimpleHandler) Handle(_ context.Context, r slog.Record) error {
	level := r.Level.String()

	timeStr := r.Time.Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("%s [%s] %s", timeStr, level, r.Message)

	r.Attrs(func(a slog.Attr) bool {
		msg += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	_, err := fmt.Fprintln(h.Output, msg)
	return err
}

func (h *SimpleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *SimpleHandler) WithGroup(name string) slog.Handler {
	return h
}

// ParseLevel parses a string into a slog.Level.
func ParseLevel(levelStr string) (slog.Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level '%s'", levelStr)
	}
}
