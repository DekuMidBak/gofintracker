package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DekuMidBak/gofintracker/internal/transaction"
	kafkago "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:     kafkago.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafkago.LeastBytes{},
		},
	}
}

func (p *Producer) PublishTransactionCreated(
	ctx context.Context,
	event transaction.TransactionCreatedEvent,
) error {
	payload, err := EncodeTransactionCreated(event)
	if err != nil {
		return err
	}

	message := kafkago.Message{
		Key:   []byte(event.UserID),
		Value: payload,
		Time:  event.CreatedAt,
	}
	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("write transaction created event: %w", err)
	}

	return nil
}

func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("close kafka writer: %w", err)
	}

	return nil
}

func EncodeTransactionCreated(event transaction.TransactionCreatedEvent) ([]byte, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal transaction created event: %w", err)
	}

	return payload, nil
}
