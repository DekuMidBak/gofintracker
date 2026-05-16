package analytics

import (
	"context"
	"strings"
	"time"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ProcessTransactionCreated(ctx context.Context, event TransactionCreated) (bool, error) {
	normalized, err := normalizeTransactionCreated(event)
	if err != nil {
		return false, err
	}

	return s.repository.ApplyTransactionCreated(ctx, normalized)
}

func (s *Service) GetMonthlySummary(
	ctx context.Context,
	userID string,
	year int,
	month int,
) ([]MonthlySummary, error) {
	normalizedUserID, err := normalizeID(userID)
	if err != nil {
		return nil, err
	}

	if !validPeriod(year, month) {
		return nil, ErrInvalidPeriod
	}

	return s.repository.GetMonthlySummary(ctx, normalizedUserID, year, month)
}

func (s *Service) GetCategoryStats(
	ctx context.Context,
	filter CategoryStatsFilter,
) ([]CategoryStat, error) {
	userID, err := normalizeID(filter.UserID)
	if err != nil {
		return nil, err
	}

	if !validPeriod(filter.Year, filter.Month) {
		return nil, ErrInvalidPeriod
	}

	if filter.Type != nil && !filter.Type.Valid() {
		return nil, ErrInvalidType
	}

	filter.UserID = userID
	return s.repository.GetCategoryStats(ctx, filter)
}

func (t TransactionType) Valid() bool {
	return t == TransactionTypeIncome || t == TransactionTypeExpense
}

func normalizeTransactionCreated(event TransactionCreated) (TransactionCreated, error) {
	eventID, err := normalizeID(event.EventID)
	if err != nil {
		return TransactionCreated{}, err
	}

	userID, err := normalizeID(event.UserID)
	if err != nil {
		return TransactionCreated{}, err
	}

	transactionID, err := normalizeID(event.TransactionID)
	if err != nil {
		return TransactionCreated{}, err
	}

	categoryID, err := normalizeID(event.CategoryID)
	if err != nil {
		return TransactionCreated{}, err
	}

	if !event.Type.Valid() {
		return TransactionCreated{}, ErrInvalidType
	}

	if event.Amount <= 0 {
		return TransactionCreated{}, ErrInvalidAmount
	}

	currency, err := normalizeCurrency(event.Currency)
	if err != nil {
		return TransactionCreated{}, err
	}

	if event.OccurredAt.IsZero() || event.OccurredAt.Year() < 1970 {
		return TransactionCreated{}, ErrInvalidPeriod
	}

	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	event.EventID = eventID
	event.UserID = userID
	event.TransactionID = transactionID
	event.CategoryID = categoryID
	event.Currency = currency
	event.OccurredAt = event.OccurredAt.UTC()
	event.CreatedAt = event.CreatedAt.UTC()

	return event, nil
}

func normalizeID(id string) (string, error) {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		return "", ErrInvalidID
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

func validPeriod(year int, month int) bool {
	return year >= 1970 && month >= 1 && month <= 12
}
