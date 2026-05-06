package logging

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

const (
	FormatJSON = "json"
	FormatText = "text"
)

type Config struct {
	Level  string
	Format string
}

func New(out io.Writer, cfg Config) (*slog.Logger, error) {
	level, err := ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	if format == "" {
		format = FormatJSON
	}

	var handler slog.Handler
	switch format {
	case FormatJSON:
		handler = slog.NewJSONHandler(out, opts)
	case FormatText:
		handler = slog.NewTextHandler(out, opts)
	default:
		return nil, fmt.Errorf("unsupported log format %q", cfg.Format)
	}

	return slog.New(handler), nil
}

func ParseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unsupported log level %q", value)
	}
}
