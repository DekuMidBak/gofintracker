package analytics

import (
	"context"
	"errors"
)

var (
	ErrInvalidID       = errors.New("invalid id")
	ErrInvalidPeriod   = errors.New("invalid period")
	ErrInvalidType     = errors.New("invalid transaction type")
	ErrInvalidAmount   = errors.New("invalid amount")
	ErrInvalidCurrency = errors.New("invalid currency")
)

type CategoryStatsFilter struct {
	UserID string
	Year   int
	Month  int
	Type   *TransactionType
}

type Repository interface {
	ApplyTransactionCreated(ctx context.Context, event TransactionCreated) (applied bool, err error)
	GetMonthlySummary(ctx context.Context, userID string, year int, month int) ([]MonthlySummary, error)
	GetCategoryStats(ctx context.Context, filter CategoryStatsFilter) ([]CategoryStat, error)
}
