package transaction

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceCreateCategoryValidatesAndNormalizesInput(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	category, err := service.CreateCategory(context.Background(), CreateCategoryParams{
		UserID: " user-1 ",
		Name:   " Food ",
		Type:   TypeExpense,
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	if category.Name != "Food" {
		t.Fatalf("expected trimmed category name, got %q", category.Name)
	}

	if repository.createCategoryParams.UserID != "user-1" {
		t.Fatalf("expected trimmed user id, got %q", repository.createCategoryParams.UserID)
	}
}

func TestServiceCreateCategoryRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})

	tests := []struct {
		name   string
		params CreateCategoryParams
		want   error
	}{
		{
			name: "empty user id",
			params: CreateCategoryParams{
				Name: "Food",
				Type: TypeExpense,
			},
			want: ErrInvalidUserID,
		},
		{
			name: "empty name",
			params: CreateCategoryParams{
				UserID: "user-1",
				Type:   TypeExpense,
			},
			want: ErrInvalidCategoryName,
		},
		{
			name: "invalid type",
			params: CreateCategoryParams{
				UserID: "user-1",
				Name:   "Food",
				Type:   Type("other"),
			},
			want: ErrInvalidType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateCategory(context.Background(), tt.params)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestServiceCreateTransactionValidatesAndNormalizesInput(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	occurredAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	created, err := service.CreateTransaction(context.Background(), CreateTransactionParams{
		UserID:      " user-1 ",
		CategoryID:  " category-1 ",
		Type:        TypeExpense,
		Amount:      1200,
		Currency:    " rub ",
		Description: " lunch ",
		OccurredAt:  occurredAt,
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	if created.Currency != "RUB" {
		t.Fatalf("expected normalized currency, got %q", created.Currency)
	}

	if repository.createTransactionParams.Description != "lunch" {
		t.Fatalf("expected trimmed description, got %q", repository.createTransactionParams.Description)
	}
}

func TestServiceCreateTransactionRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	valid := CreateTransactionParams{
		UserID:     "user-1",
		CategoryID: "category-1",
		Type:       TypeExpense,
		Amount:     1200,
		Currency:   "RUB",
		OccurredAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	tests := []struct {
		name   string
		mutate func(*CreateTransactionParams)
		want   error
	}{
		{
			name: "empty user id",
			mutate: func(params *CreateTransactionParams) {
				params.UserID = ""
			},
			want: ErrInvalidUserID,
		},
		{
			name: "empty category id",
			mutate: func(params *CreateTransactionParams) {
				params.CategoryID = ""
			},
			want: ErrInvalidCategoryID,
		},
		{
			name: "invalid type",
			mutate: func(params *CreateTransactionParams) {
				params.Type = Type("other")
			},
			want: ErrInvalidType,
		},
		{
			name: "non-positive amount",
			mutate: func(params *CreateTransactionParams) {
				params.Amount = 0
			},
			want: ErrInvalidAmount,
		},
		{
			name: "invalid currency",
			mutate: func(params *CreateTransactionParams) {
				params.Currency = "RU"
			},
			want: ErrInvalidCurrency,
		},
		{
			name: "empty occurred at",
			mutate: func(params *CreateTransactionParams) {
				params.OccurredAt = time.Time{}
			},
			want: ErrInvalidOccurredAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := valid
			tt.mutate(&params)

			_, err := service.CreateTransaction(context.Background(), params)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestServiceListTransactionsNormalizesFilter(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	categoryID := " category-1 "

	_, err := service.ListTransactions(context.Background(), ListTransactionsFilter{
		UserID:     " user-1 ",
		CategoryID: &categoryID,
	})
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}

	if repository.listTransactionsFilter.UserID != "user-1" {
		t.Fatalf("expected trimmed user id, got %q", repository.listTransactionsFilter.UserID)
	}

	if repository.listTransactionsFilter.CategoryID == nil || *repository.listTransactionsFilter.CategoryID != "category-1" {
		t.Fatalf("expected trimmed category id, got %+v", repository.listTransactionsFilter.CategoryID)
	}

	if repository.listTransactionsFilter.Limit != DefaultListLimit {
		t.Fatalf("expected default limit, got %d", repository.listTransactionsFilter.Limit)
	}
}

func TestServiceListTransactionsRejectsInvalidFilter(t *testing.T) {
	service := NewService(&fakeRepository{})
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		filter ListTransactionsFilter
		want   error
	}{
		{
			name: "empty user id",
			filter: ListTransactionsFilter{
				Limit: 10,
			},
			want: ErrInvalidUserID,
		},
		{
			name: "invalid type",
			filter: ListTransactionsFilter{
				UserID: "user-1",
				Type:   typePtr(Type("other")),
			},
			want: ErrInvalidType,
		},
		{
			name: "invalid time range",
			filter: ListTransactionsFilter{
				UserID: "user-1",
				From:   &from,
				To:     &to,
			},
			want: ErrInvalidTimeRange,
		},
		{
			name: "negative offset",
			filter: ListTransactionsFilter{
				UserID: "user-1",
				Offset: -1,
			},
			want: ErrInvalidPagination,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ListTransactions(context.Background(), tt.filter)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestServiceGetBalanceRejectsEmptyUserID(t *testing.T) {
	service := NewService(&fakeRepository{})

	_, err := service.GetBalance(context.Background(), " ")
	if !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func typePtr(transactionType Type) *Type {
	return &transactionType
}

type fakeRepository struct {
	createCategoryParams    CreateCategoryParams
	createTransactionParams CreateTransactionParams
	listTransactionsFilter  ListTransactionsFilter
}

func (r *fakeRepository) CreateCategory(_ context.Context, params CreateCategoryParams) (Category, error) {
	r.createCategoryParams = params

	return Category{
		ID:        "category-1",
		UserID:    params.UserID,
		Name:      params.Name,
		Type:      params.Type,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}, nil
}

func (r *fakeRepository) ListCategories(_ context.Context, filter ListCategoriesFilter) ([]Category, error) {
	return []Category{
		{
			ID:     "category-1",
			UserID: filter.UserID,
			Name:   "Food",
			Type:   TypeExpense,
		},
	}, nil
}

func (r *fakeRepository) CreateTransaction(_ context.Context, params CreateTransactionParams) (Transaction, error) {
	r.createTransactionParams = params

	return Transaction{
		ID:          "transaction-1",
		UserID:      params.UserID,
		CategoryID:  params.CategoryID,
		Type:        params.Type,
		Amount:      params.Amount,
		Currency:    params.Currency,
		Description: params.Description,
		OccurredAt:  params.OccurredAt,
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}, nil
}

func (r *fakeRepository) ListTransactions(_ context.Context, filter ListTransactionsFilter) (ListTransactionsResult, error) {
	r.listTransactionsFilter = filter

	return ListTransactionsResult{
		Transactions: []Transaction{
			{
				ID:     "transaction-1",
				UserID: filter.UserID,
			},
		},
		TotalCount: 1,
	}, nil
}

func (r *fakeRepository) GetBalance(context.Context, string) ([]CurrencyBalance, error) {
	return []CurrencyBalance{
		{
			Currency:      "RUB",
			IncomeAmount:  100_000,
			ExpenseAmount: 15_000,
			BalanceAmount: 85_000,
		},
	}, nil
}
