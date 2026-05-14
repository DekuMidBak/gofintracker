package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	transactionv1 "github.com/DekuMidBak/gofintracker/gen/go/transaction/v1"
	"github.com/DekuMidBak/gofintracker/internal/transaction"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestServerCreateCategory(t *testing.T) {
	service := &fakeService{
		createCategoryResult: transaction.Category{
			ID:        "category-1",
			UserID:    "user-1",
			Name:      "Food",
			Type:      transaction.TypeExpense,
			CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		},
	}
	server := NewServer(service)

	resp, err := server.CreateCategory(context.Background(), &transactionv1.CreateCategoryRequest{
		UserId: "user-1",
		Name:   "Food",
		Type:   transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE,
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	if service.createCategoryParams.Type != transaction.TypeExpense {
		t.Fatalf("expected expense type, got %q", service.createCategoryParams.Type)
	}

	if resp.GetCategory().GetId() != "category-1" {
		t.Fatalf("expected category id, got %q", resp.GetCategory().GetId())
	}

	if resp.GetCategory().GetType() != transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE {
		t.Fatalf("expected expense proto type, got %s", resp.GetCategory().GetType())
	}
}

func TestServerListTransactionsForwardsOptionalFilters(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)
	categoryID := "category-1"
	transactionType := transactionv1.TransactionType_TRANSACTION_TYPE_INCOME
	service := &fakeService{
		listTransactionsResult: transaction.ListTransactionsResult{
			Transactions: []transaction.Transaction{
				{
					ID:         "transaction-1",
					UserID:     "user-1",
					CategoryID: categoryID,
					Type:       transaction.TypeIncome,
					Amount:     100_000,
					Currency:   "RUB",
					OccurredAt: from,
					CreatedAt:  from,
				},
			},
			TotalCount: 1,
		},
	}
	server := NewServer(service)

	resp, err := server.ListTransactions(context.Background(), &transactionv1.ListTransactionsRequest{
		UserId:     "user-1",
		From:       timestamppb.New(from),
		To:         timestamppb.New(to),
		CategoryId: &categoryID,
		Type:       &transactionType,
		Limit:      20,
		Offset:     5,
	})
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}

	filter := service.listTransactionsFilter
	if filter.From == nil || !filter.From.Equal(from) {
		t.Fatalf("expected from filter, got %+v", filter.From)
	}

	if filter.To == nil || !filter.To.Equal(to) {
		t.Fatalf("expected to filter, got %+v", filter.To)
	}

	if filter.CategoryID == nil || *filter.CategoryID != categoryID {
		t.Fatalf("expected category filter, got %+v", filter.CategoryID)
	}

	if filter.Type == nil || *filter.Type != transaction.TypeIncome {
		t.Fatalf("expected income type filter, got %+v", filter.Type)
	}

	if filter.Limit != 20 || filter.Offset != 5 {
		t.Fatalf("expected pagination to be forwarded, got limit=%d offset=%d", filter.Limit, filter.Offset)
	}

	if resp.GetTotalCount() != 1 || len(resp.GetTransactions()) != 1 {
		t.Fatalf("expected one transaction, got %+v", resp)
	}
}

func TestServerGetBalance(t *testing.T) {
	service := &fakeService{
		balances: []transaction.CurrencyBalance{
			{
				Currency:      "RUB",
				IncomeAmount:  100_000,
				ExpenseAmount: 15_000,
				BalanceAmount: 85_000,
			},
		},
	}
	server := NewServer(service)

	resp, err := server.GetBalance(context.Background(), &transactionv1.GetBalanceRequest{
		UserId: "user-1",
	})
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	if service.balanceUserID != "user-1" {
		t.Fatalf("expected user id to be forwarded")
	}

	if len(resp.GetBalances()) != 1 || resp.GetBalances()[0].GetBalanceAmount() != 85_000 {
		t.Fatalf("expected balance response, got %+v", resp.GetBalances())
	}
}

func TestServerMapsValidationErrorToInvalidArgument(t *testing.T) {
	server := NewServer(&fakeService{
		createTransactionErr: transaction.ErrInvalidAmount,
	})

	_, err := server.CreateTransaction(context.Background(), &transactionv1.CreateTransactionRequest{
		UserId:     "user-1",
		CategoryId: "category-1",
		Type:       transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		Amount:     0,
		Currency:   "RUB",
		OccurredAt: timestamppb.Now(),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %s", status.Code(err))
	}
}

func TestServerMapsKnownRepositoryErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{
			name: "duplicate category",
			err:  transaction.ErrDuplicateCategory,
			code: codes.AlreadyExists,
		},
		{
			name: "not found",
			err:  transaction.ErrNotFound,
			code: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(&fakeService{
				createCategoryErr: tt.err,
			})

			_, err := server.CreateCategory(context.Background(), &transactionv1.CreateCategoryRequest{})
			if status.Code(err) != tt.code {
				t.Fatalf("expected %s, got %s", tt.code, status.Code(err))
			}
		})
	}
}

func TestServerMapsUnknownErrorToInternal(t *testing.T) {
	server := NewServer(&fakeService{
		createCategoryErr: errors.New("boom"),
	})

	_, err := server.CreateCategory(context.Background(), &transactionv1.CreateCategoryRequest{})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal, got %s", status.Code(err))
	}
}

type fakeService struct {
	createCategoryParams transaction.CreateCategoryParams
	createCategoryResult transaction.Category
	createCategoryErr    error

	listCategoriesFilter transaction.ListCategoriesFilter
	categories           []transaction.Category
	listCategoriesErr    error

	createTransactionParams transaction.CreateTransactionParams
	createTransactionResult transaction.Transaction
	createTransactionErr    error

	listTransactionsFilter transaction.ListTransactionsFilter
	listTransactionsResult transaction.ListTransactionsResult
	listTransactionsErr    error

	balanceUserID string
	balances      []transaction.CurrencyBalance
	balanceErr    error
}

func (s *fakeService) CreateCategory(
	_ context.Context,
	params transaction.CreateCategoryParams,
) (transaction.Category, error) {
	s.createCategoryParams = params
	if s.createCategoryErr != nil {
		return transaction.Category{}, s.createCategoryErr
	}

	return s.createCategoryResult, nil
}

func (s *fakeService) ListCategories(
	_ context.Context,
	filter transaction.ListCategoriesFilter,
) ([]transaction.Category, error) {
	s.listCategoriesFilter = filter
	if s.listCategoriesErr != nil {
		return nil, s.listCategoriesErr
	}

	return s.categories, nil
}

func (s *fakeService) CreateTransaction(
	_ context.Context,
	params transaction.CreateTransactionParams,
) (transaction.Transaction, error) {
	s.createTransactionParams = params
	if s.createTransactionErr != nil {
		return transaction.Transaction{}, s.createTransactionErr
	}

	return s.createTransactionResult, nil
}

func (s *fakeService) ListTransactions(
	_ context.Context,
	filter transaction.ListTransactionsFilter,
) (transaction.ListTransactionsResult, error) {
	s.listTransactionsFilter = filter
	if s.listTransactionsErr != nil {
		return transaction.ListTransactionsResult{}, s.listTransactionsErr
	}

	return s.listTransactionsResult, nil
}

func (s *fakeService) GetBalance(_ context.Context, userID string) ([]transaction.CurrencyBalance, error) {
	s.balanceUserID = userID
	if s.balanceErr != nil {
		return nil, s.balanceErr
	}

	return s.balances, nil
}
