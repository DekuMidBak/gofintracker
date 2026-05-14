package transaction

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTransactionCreatedEventJSON(t *testing.T) {
	event := TransactionCreatedEvent{
		EventID:       "event-1",
		UserID:        "user-1",
		TransactionID: "transaction-1",
		Type:          TypeExpense,
		Amount:        1500,
		Currency:      "RUB",
		CategoryID:    "category-1",
		OccurredAt:    time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		CreatedAt:     time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	if decoded["event_id"] != "event-1" {
		t.Fatalf("expected event_id field, got %q", decoded["event_id"])
	}

	if decoded["transaction_id"] != "transaction-1" {
		t.Fatalf("expected transaction_id field, got %q", decoded["transaction_id"])
	}

	if decoded["type"] != "expense" {
		t.Fatalf("expected expense type, got %q", decoded["type"])
	}

	if decoded["amount"] != float64(1500) {
		t.Fatalf("expected amount field, got %v", decoded["amount"])
	}
}
