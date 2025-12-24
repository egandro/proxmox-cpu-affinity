package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSimpleHandler_Enabled(t *testing.T) {
	h := &SimpleHandler{Level: slog.LevelInfo}
	ctx := context.Background()

	assert.False(t, h.Enabled(ctx, slog.LevelDebug))
	assert.True(t, h.Enabled(ctx, slog.LevelInfo))
	assert.True(t, h.Enabled(ctx, slog.LevelWarn))
	assert.True(t, h.Enabled(ctx, slog.LevelError))
}

func TestSimpleHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	h := &SimpleHandler{Output: &buf, Level: slog.LevelInfo}
	ctx := context.Background()

	// Use a fixed time for reproducible output
	fixedTime := time.Date(2023, 10, 27, 10, 0, 0, 0, time.UTC)

	r := slog.NewRecord(fixedTime, slog.LevelInfo, "test message", 0)
	r.AddAttrs(slog.String("key", "value"), slog.Int("count", 42))

	err := h.Handle(ctx, r)
	assert.NoError(t, err)

	output := buf.String()
	// Expected format: "2006-01-02 15:04:05 [LEVEL] Message key=value count=42\n"
	expected := "2023-10-27 10:00:00 [INFO] test message key=value count=42\n"

	assert.Equal(t, expected, output)
}

func TestSimpleHandler_WithAttrs(t *testing.T) {
	h := &SimpleHandler{Level: slog.LevelInfo}
	newH := h.WithAttrs([]slog.Attr{slog.String("a", "b")})
	assert.Equal(t, h, newH, "WithAttrs should currently be a no-op returning the same handler")
}

func TestSimpleHandler_WithGroup(t *testing.T) {
	h := &SimpleHandler{Level: slog.LevelInfo}
	newH := h.WithGroup("group")
	assert.Equal(t, h, newH, "WithGroup should currently be a no-op returning the same handler")
}
