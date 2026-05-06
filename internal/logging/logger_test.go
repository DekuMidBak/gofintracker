package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevelSupportsKnownLevels(t *testing.T) {
	tests := map[string]slog.Level{
		"":        slog.LevelInfo,
		"debug":   slog.LevelDebug,
		"info":    slog.LevelInfo,
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := ParseLevel(input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if got != want {
				t.Fatalf("expected %s, got %s", want, got)
			}
		})
	}
}

func TestParseLevelReturnsErrorForUnknownLevel(t *testing.T) {
	_, err := ParseLevel("trace")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewCreatesJSONLoggerByDefault(t *testing.T) {
	var out bytes.Buffer

	logger, err := New(&out, Config{Level: "debug"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	logger.Debug("hello")

	got := out.String()
	if !strings.Contains(got, `"level":"DEBUG"`) {
		t.Fatalf("expected JSON debug log, got %q", got)
	}
}

func TestNewCreatesTextLogger(t *testing.T) {
	var out bytes.Buffer

	logger, err := New(&out, Config{Level: "info", Format: FormatText})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	logger.Info("hello")

	got := out.String()
	if !strings.Contains(got, "level=INFO") {
		t.Fatalf("expected text info log, got %q", got)
	}
}

func TestNewReturnsErrorForUnsupportedFormat(t *testing.T) {
	var out bytes.Buffer

	_, err := New(&out, Config{Format: "xml"})
	if err == nil {
		t.Fatal("expected error")
	}
}
