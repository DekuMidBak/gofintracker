package transaction

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound          = errors.New("transaction resource not found")
	ErrDuplicateCategory = errors.New("category already exists")
	ErrInvalidID         = errors.New("invalid id")
)

type CreateCategoryParams struct {
	UserID string
	Name   string
	Type   Type
}

type ListCategoriesFilter struct {
	UserID string
	Type   *Type
}

type CreateTransactionParams struct {
	UserID      string
	CategoryID  string
	Type        Type
	Amount      int64
	Currency    string
	Description string
	OccurredAt  time.Time
}

type ListTransactionsFilter struct {
	UserID     string
	From       *time.Time
	To         *time.Time
	CategoryID *string
	Type       *Type
	Limit      int
	Offset     int
}

type ListTransactionsResult struct {
	Transactions []Transaction
	TotalCount   int
}

type Repository interface {
	CreateCategory(ctx context.Context, params CreateCategoryParams) (Category, error)
	ListCategories(ctx context.Context, filter ListCategoriesFilter) ([]Category, error)
	CreateTransaction(ctx context.Context, params CreateTransactionParams) (Transaction, error)
	ListTransactions(ctx context.Context, filter ListTransactionsFilter) (ListTransactionsResult, error)
	GetBalance(ctx context.Context, userID string) ([]CurrencyBalance, error)
}
