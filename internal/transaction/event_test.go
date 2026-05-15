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

func TestNewTransactionCreatedEvent(t *testing.T) {
	created := Transaction{
		ID:         "transaction-1",
		UserID:     "user-1",
		CategoryID: "category-1",
		Type:       TypeIncome,
		Amount:     100_000,
		Currency:   "RUB",
		OccurredAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		CreatedAt:  time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
	}

	event, err := NewTransactionCreatedEvent(created)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	if event.EventID == "" {
		t.Fatal("expected generated event id")
	}

	if event.TransactionID != created.ID {
		t.Fatalf("expected transaction id %q, got %q", created.ID, event.TransactionID)
	}

	if event.UserID != created.UserID {
		t.Fatalf("expected user id %q, got %q", created.UserID, event.UserID)
	}
}
