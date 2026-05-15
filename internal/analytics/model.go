package analytics

import "time"

type TransactionType string

const (
	TransactionTypeIncome  TransactionType = "income"
	TransactionTypeExpense TransactionType = "expense"
)

type TransactionCreated struct {
	EventID       string
	UserID        string
	TransactionID string
	Type          TransactionType
	Amount        int64
	Currency      string
	CategoryID    string
	OccurredAt    time.Time
	CreatedAt     time.Time
}

type MonthlySummary struct {
	Currency      string
	IncomeAmount  int64
	ExpenseAmount int64
	BalanceAmount int64
}

type CategoryStat struct {
	CategoryID string
	Currency   string
	Type       TransactionType
	Amount     int64
}
