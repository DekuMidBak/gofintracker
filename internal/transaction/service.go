package transaction

import (
	"context"
	"errors"
	"log/slog"
	"strings"
)

const (
	DefaultListLimit = 50
	MaxListLimit     = 100
)

var (
	ErrInvalidUserID       = errors.New("invalid user id")
	ErrInvalidCategoryID   = errors.New("invalid category id")
	ErrInvalidCategoryName = errors.New("invalid category name")
	ErrInvalidType         = errors.New("invalid transaction type")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrInvalidCurrency     = errors.New("invalid currency")
	ErrInvalidOccurredAt   = errors.New("invalid occurred_at")
	ErrInvalidTimeRange    = errors.New("invalid time range")
	ErrInvalidPagination   = errors.New("invalid pagination")
)

type Service struct {
	repository     Repository
	eventPublisher EventPublisher
	logger         *slog.Logger
}

func NewService(repository Repository) *Service {
	return NewServiceWithEvents(repository, NoopEventPublisher{}, slog.Default())
}

func NewServiceWithEvents(repository Repository, eventPublisher EventPublisher, logger *slog.Logger) *Service {
	if eventPublisher == nil {
		eventPublisher = NoopEventPublisher{}
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		repository:     repository,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

func (s *Service) CreateCategory(ctx context.Context, params CreateCategoryParams) (Category, error) {
	userID, err := normalizeID(params.UserID, ErrInvalidUserID)
	if err != nil {
		return Category{}, err
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return Category{}, ErrInvalidCategoryName
	}

	if !params.Type.Valid() {
		return Category{}, ErrInvalidType
	}

	return s.repository.CreateCategory(ctx, CreateCategoryParams{
		UserID: userID,
		Name:   name,
		Type:   params.Type,
	})
}

func (s *Service) ListCategories(ctx context.Context, filter ListCategoriesFilter) ([]Category, error) {
	userID, err := normalizeID(filter.UserID, ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	if filter.Type != nil && !filter.Type.Valid() {
		return nil, ErrInvalidType
	}

	return s.repository.ListCategories(ctx, ListCategoriesFilter{
		UserID: userID,
		Type:   filter.Type,
	})
}

func (s *Service) CreateTransaction(ctx context.Context, params CreateTransactionParams) (Transaction, error) {
	userID, err := normalizeID(params.UserID, ErrInvalidUserID)
	if err != nil {
		return Transaction{}, err
	}

	categoryID, err := normalizeID(params.CategoryID, ErrInvalidCategoryID)
	if err != nil {
		return Transaction{}, err
	}

	if !params.Type.Valid() {
		return Transaction{}, ErrInvalidType
	}

	if params.Amount <= 0 {
		return Transaction{}, ErrInvalidAmount
	}

	currency, err := normalizeCurrency(params.Currency)
	if err != nil {
		return Transaction{}, err
	}

	if params.OccurredAt.IsZero() {
		return Transaction{}, ErrInvalidOccurredAt
	}

	created, err := s.repository.CreateTransaction(ctx, CreateTransactionParams{
		UserID:      userID,
		CategoryID:  categoryID,
		Type:        params.Type,
		Amount:      params.Amount,
		Currency:    currency,
		Description: strings.TrimSpace(params.Description),
		OccurredAt:  params.OccurredAt,
	})
	if err != nil {
		return Transaction{}, err
	}

	s.publishTransactionCreated(ctx, created)

	return created, nil
}

func (s *Service) ListTransactions(ctx context.Context, filter ListTransactionsFilter) (ListTransactionsResult, error) {
	userID, err := normalizeID(filter.UserID, ErrInvalidUserID)
	if err != nil {
		return ListTransactionsResult{}, err
	}

	if filter.CategoryID != nil {
		categoryID, err := normalizeID(*filter.CategoryID, ErrInvalidCategoryID)
		if err != nil {
			return ListTransactionsResult{}, err
		}

		filter.CategoryID = &categoryID
	}

	if filter.Type != nil && !filter.Type.Valid() {
		return ListTransactionsResult{}, ErrInvalidType
	}

	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return ListTransactionsResult{}, ErrInvalidTimeRange
	}

	limit, err := normalizeLimit(filter.Limit)
	if err != nil {
		return ListTransactionsResult{}, err
	}

	if filter.Offset < 0 {
		return ListTransactionsResult{}, ErrInvalidPagination
	}

	filter.UserID = userID
	filter.Limit = limit

	return s.repository.ListTransactions(ctx, filter)
}

func (s *Service) GetBalance(ctx context.Context, userID string) ([]CurrencyBalance, error) {
	normalizedUserID, err := normalizeID(userID, ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	return s.repository.GetBalance(ctx, normalizedUserID)
}

func (t Type) Valid() bool {
	return t == TypeIncome || t == TypeExpense
}

func normalizeID(id string, invalidErr error) (string, error) {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		return "", invalidErr
	}

	return normalized, nil
}

func normalizeCurrency(currency string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(currency))
	if len(normalized) != 3 {
		return "", ErrInvalidCurrency
	}

	for _, char := range normalized {
		if char < 'A' || char > 'Z' {
			return "", ErrInvalidCurrency
		}
	}

	return normalized, nil
}

func normalizeLimit(limit int) (int, error) {
	if limit < 0 {
		return 0, ErrInvalidPagination
	}

	if limit == 0 {
		return DefaultListLimit, nil
	}

	if limit > MaxListLimit {
		return MaxListLimit, nil
	}

	return limit, nil
}

func (s *Service) publishTransactionCreated(ctx context.Context, created Transaction) {
	event, err := NewTransactionCreatedEvent(created)
	if err != nil {
		s.logger.Warn("failed to build transaction created event", "error", err, "transaction_id", created.ID)
		return
	}

	if err := s.eventPublisher.PublishTransactionCreated(ctx, event); err != nil {
		s.logger.Warn(
			"failed to publish transaction created event",
			"error", err,
			"event_id", event.EventID,
			"transaction_id", event.TransactionID,
		)
	}
}
