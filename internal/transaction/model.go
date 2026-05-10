package transaction

import "time"

type Type string

const (
	TypeIncome  Type = "income"
	TypeExpense Type = "expense"
)

type Category struct {
	ID        string
	UserID    string
	Name      string
	Type      Type
	CreatedAt time.Time
}

type Transaction struct {
	ID          string
	UserID      string
	CategoryID  string
	Type        Type
	Amount      int64
	Currency    string
	Description string
	OccurredAt  time.Time
	CreatedAt   time.Time
}

type CurrencyBalance struct {
	Currency      string
	IncomeAmount  int64
	ExpenseAmount int64
	BalanceAmount int64
}
