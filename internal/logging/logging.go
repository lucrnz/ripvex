package logging

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
)

type ctxKey struct{}

// New constructs a slog.Logger with the given level and format writing to stderr.
func New(level, format string) (*slog.Logger, error) {
	lvl, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: lvl}
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	case "text", "":
		handler = slog.NewTextHandler(os.Stderr, opts)
	default:
		return nil, errors.New("unsupported log format: " + format)
	}

	return slog.New(handler), nil
}

// WithContext attaches a logger to the context.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the logger stored in context or a default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

func parseLevel(level string) (*slog.LevelVar, error) {
	lv := new(slog.LevelVar)
	lower := strings.ToLower(level)
	if lower == "" {
		lower = "info"
	}
	if err := lv.UnmarshalText([]byte(lower)); err != nil {
		return nil, err
	}
	return lv, nil
}
