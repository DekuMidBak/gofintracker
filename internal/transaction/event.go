package transaction

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

func NewTransactionCreatedEvent(item Transaction) (TransactionCreatedEvent, error) {
	eventID, err := newEventID()
	if err != nil {
		return TransactionCreatedEvent{}, err
	}

	return TransactionCreatedEvent{
		EventID:       eventID,
		UserID:        item.UserID,
		TransactionID: item.ID,
		Type:          item.Type,
		Amount:        item.Amount,
		Currency:      item.Currency,
		CategoryID:    item.CategoryID,
		OccurredAt:    item.OccurredAt,
		CreatedAt:     item.CreatedAt,
	}, nil
}

type NoopEventPublisher struct{}

func (NoopEventPublisher) PublishTransactionCreated(context.Context, TransactionCreatedEvent) error {
	return nil
}

func newEventID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate event id: %w", err)
	}

	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80

	return hex.EncodeToString(bytes[0:4]) + "-" +
		hex.EncodeToString(bytes[4:6]) + "-" +
		hex.EncodeToString(bytes[6:8]) + "-" +
		hex.EncodeToString(bytes[8:10]) + "-" +
		hex.EncodeToString(bytes[10:16]), nil
}
