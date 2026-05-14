package transaction

import (
	"context"
	"time"
)

const TransactionCreatedEventName = "transaction.created"

type TransactionCreatedEvent struct {
	EventID       string    `json:"event_id"`
	UserID        string    `json:"user_id"`
	TransactionID string    `json:"transaction_id"`
	Type          Type      `json:"type"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	CategoryID    string    `json:"category_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type EventPublisher interface {
	PublishTransactionCreated(ctx context.Context, event TransactionCreatedEvent) error
}

type NoopEventPublisher struct{}

func (NoopEventPublisher) PublishTransactionCreated(context.Context, TransactionCreatedEvent) error {
	return nil
}
