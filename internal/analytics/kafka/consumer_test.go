package kafka

import (
	"testing"
	"time"

	"github.com/DekuMidBak/gofintracker/internal/analytics"
)

func TestDecodeTransactionCreated(t *testing.T) {
	payload := []byte(`{
		"event_id":"event-1",
		"user_id":"user-1",
		"transaction_id":"transaction-1",
		"type":"expense",
		"amount":1500,
		"currency":"RUB",
		"category_id":"category-1",
		"occurred_at":"2026-01-02T03:04:05Z",
		"created_at":"2026-01-02T03:04:06Z"
	}`)

	event, err := DecodeTransactionCreated(payload)
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}

	if event.EventID != "event-1" {
		t.Fatalf("expected event id, got %q", event.EventID)
	}

	if event.Type != analytics.TransactionTypeExpense {
		t.Fatalf("expected expense type, got %q", event.Type)
	}

	if event.Amount != 1500 {
		t.Fatalf("expected amount 1500, got %d", event.Amount)
	}

	expectedOccurredAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if !event.OccurredAt.Equal(expectedOccurredAt) {
		t.Fatalf("expected occurred_at %s, got %s", expectedOccurredAt, event.OccurredAt)
	}
}

func TestDecodeTransactionCreatedReturnsErrorForInvalidJSON(t *testing.T) {
	_, err := DecodeTransactionCreated([]byte(`{`))
	if err == nil {
		t.Fatal("expected decode error")
	}
}
