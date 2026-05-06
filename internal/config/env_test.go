package config

import (
	"testing"
	"time"
)

func TestStringReturnsFallbackWhenUnset(t *testing.T) {
	t.Setenv("GOFINTRACKER_TEST_STRING", "")

	got := String("GOFINTRACKER_TEST_STRING", "fallback")
	if got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestStringReturnsEnvValue(t *testing.T) {
	t.Setenv("GOFINTRACKER_TEST_STRING", "value")

	got := String("GOFINTRACKER_TEST_STRING", "fallback")
	if got != "value" {
		t.Fatalf("expected env value, got %q", got)
	}
}

func TestRequiredStringReturnsErrorWhenUnset(t *testing.T) {
	t.Setenv("GOFINTRACKER_TEST_REQUIRED", "")

	_, err := RequiredString("GOFINTRACKER_TEST_REQUIRED")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRequiredStringReturnsEnvValue(t *testing.T) {
	t.Setenv("GOFINTRACKER_TEST_REQUIRED", "value")

	got, err := RequiredString("GOFINTRACKER_TEST_REQUIRED")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got != "value" {
		t.Fatalf("expected env value, got %q", got)
	}
}

func TestIntReturnsFallbackWhenUnset(t *testing.T) {
	t.Setenv("GOFINTRACKER_TEST_INT", "")

	got, err := Int("GOFINTRACKER_TEST_INT", 42)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got != 42 {
		t.Fatalf("expected fallback, got %d", got)
	}
}

func TestIntReturnsErrorForInvalidValue(t *testing.T) {
	t.Setenv("GOFINTRACKER_TEST_INT", "nope")

	_, err := Int("GOFINTRACKER_TEST_INT", 42)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBoolReturnsEnvValue(t *testing.T) {
	t.Setenv("GOFINTRACKER_TEST_BOOL", "true")

	got, err := Bool("GOFINTRACKER_TEST_BOOL", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !got {
		t.Fatal("expected true")
	}
}

func TestDurationReturnsEnvValue(t *testing.T) {
	t.Setenv("GOFINTRACKER_TEST_DURATION", "15m")

	got, err := Duration("GOFINTRACKER_TEST_DURATION", time.Second)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got != 15*time.Minute {
		t.Fatalf("expected 15m, got %s", got)
	}
}
