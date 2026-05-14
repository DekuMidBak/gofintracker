package kafka

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DekuMidBak/gofintracker/internal/transaction"
)

func TestEncodeTransactionCreated(t *testing.T) {
	event := transaction.TransactionCreatedEvent{
		EventID:       "event-1",
		UserID:        "user-1",
		TransactionID: "transaction-1",
		Type:          transaction.TypeIncome,
		Amount:        100_000,
		Currency:      "RUB",
		CategoryID:    "category-1",
		OccurredAt:    time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		CreatedAt:     time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
	}

	payload, err := EncodeTransactionCreated(event)
	if err != nil {
		t.Fatalf("encode event: %v", err)
	}

	var decoded transaction.TransactionCreatedEvent
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode event: %v", err)
	}

	if decoded != event {
		t.Fatalf("expected decoded event to match original: %+v", decoded)
	}
}
