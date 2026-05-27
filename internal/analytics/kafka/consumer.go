package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/DekuMidBak/gofintracker/internal/analytics"
	kafkago "github.com/segmentio/kafka-go"
)

type Processor interface {
	ProcessTransactionCreated(ctx context.Context, event analytics.TransactionCreated) (bool, error)
}

type Consumer struct {
	reader    *kafkago.Reader
	processor Processor
	logger    *slog.Logger
}

type Config struct {
	Brokers []string
	Topic   string
	GroupID string
}

func NewConsumer(config Config, processor Processor, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}

	return &Consumer{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers: config.Brokers,
			Topic:   config.Topic,
			GroupID: config.GroupID,
		}),
		processor: processor,
		logger:    logger,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}

			return fmt.Errorf("fetch transaction created event: %w", err)
		}

		if err := c.handleMessage(ctx, message); err != nil {
			c.logger.Warn("failed to handle transaction created event", "error", err)
		}
	}
}

func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("close kafka reader: %w", err)
	}

	return nil
}

func (c *Consumer) handleMessage(ctx context.Context, message kafkago.Message) error {
	event, err := DecodeTransactionCreated(message.Value)
	if err != nil {
		c.logger.Warn("skip invalid transaction created event", "error", err)
		return c.commit(ctx, message)
	}

	applied, err := c.processor.ProcessTransactionCreated(ctx, event)
	if err != nil {
		if isValidationError(err) {
			c.logger.Warn("skip invalid transaction created event", "error", err, "event_id", event.EventID)
			return c.commit(ctx, message)
		}

		return fmt.Errorf("process transaction created event %s: %w", event.EventID, err)
	}

	c.logger.Debug(
		"processed transaction created event",
		"event_id", event.EventID,
		"transaction_id", event.TransactionID,
		"applied", applied,
	)

	return c.commit(ctx, message)
}

func (c *Consumer) commit(ctx context.Context, message kafkago.Message) error {
	if err := c.reader.CommitMessages(ctx, message); err != nil {
		return fmt.Errorf("commit transaction created event: %w", err)
	}

	return nil
}

func DecodeTransactionCreated(payload []byte) (analytics.TransactionCreated, error) {
	var decoded transactionCreatedPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return analytics.TransactionCreated{}, fmt.Errorf("decode transaction created event: %w", err)
	}

	return analytics.TransactionCreated{
		EventID:       decoded.EventID,
		UserID:        decoded.UserID,
		TransactionID: decoded.TransactionID,
		Type:          analytics.TransactionType(decoded.Type),
		Amount:        decoded.Amount,
		Currency:      decoded.Currency,
		CategoryID:    decoded.CategoryID,
		OccurredAt:    decoded.OccurredAt,
		CreatedAt:     decoded.CreatedAt,
	}, nil
}

type transactionCreatedPayload struct {
	EventID       string    `json:"event_id"`
	UserID        string    `json:"user_id"`
	TransactionID string    `json:"transaction_id"`
	Type          string    `json:"type"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	CategoryID    string    `json:"category_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	CreatedAt     time.Time `json:"created_at"`
}

func isValidationError(err error) bool {
	return errors.Is(err, analytics.ErrInvalidID) ||
		errors.Is(err, analytics.ErrInvalidPeriod) ||
		errors.Is(err, analytics.ErrInvalidType) ||
		errors.Is(err, analytics.ErrInvalidAmount) ||
		errors.Is(err, analytics.ErrInvalidCurrency)
}
