package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/DekuMidBak/gofintracker/internal/transaction"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryCreateAndListCategories(t *testing.T) {
	repo, pool := newTestRepository(t)
	ctx := testContext(t)
	userID := randomUUID(t)
	t.Cleanup(func() {
		deleteUserData(t, pool, userID)
	})

	income, err := repo.CreateCategory(ctx, transaction.CreateCategoryParams{
		UserID: userID,
		Name:   "Salary",
		Type:   transaction.TypeIncome,
	})
	if err != nil {
		t.Fatalf("create income category: %v", err)
	}

	expense, err := repo.CreateCategory(ctx, transaction.CreateCategoryParams{
		UserID: userID,
		Name:   "Food",
		Type:   transaction.TypeExpense,
	})
	if err != nil {
		t.Fatalf("create expense category: %v", err)
	}

	if income.ID == "" || expense.ID == "" {
		t.Fatal("expected generated category ids")
	}

	categories, err := repo.ListCategories(ctx, transaction.ListCategoriesFilter{
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}

	if len(categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(categories))
	}

	categoryType := transaction.TypeExpense
	expenseCategories, err := repo.ListCategories(ctx, transaction.ListCategoriesFilter{
		UserID: userID,
		Type:   &categoryType,
	})
	if err != nil {
		t.Fatalf("list expense categories: %v", err)
	}

	if len(expenseCategories) != 1 || expenseCategories[0].ID != expense.ID {
		t.Fatalf("expected only expense category, got %+v", expenseCategories)
	}
}

func TestRepositoryCreateDuplicateCategoryReturnsErrDuplicateCategory(t *testing.T) {
	repo, pool := newTestRepository(t)
	ctx := testContext(t)
	userID := randomUUID(t)
	t.Cleanup(func() {
		deleteUserData(t, pool, userID)
	})

	_, err := repo.CreateCategory(ctx, transaction.CreateCategoryParams{
		UserID: userID,
		Name:   "Food",
		Type:   transaction.TypeExpense,
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	_, err = repo.CreateCategory(ctx, transaction.CreateCategoryParams{
		UserID: userID,
		Name:   "food",
		Type:   transaction.TypeExpense,
	})
	if !errors.Is(err, transaction.ErrDuplicateCategory) {
		t.Fatalf("expected ErrDuplicateCategory, got %v", err)
	}
}

func TestRepositoryCreateListTransactionsAndGetBalance(t *testing.T) {
	repo, pool := newTestRepository(t)
	ctx := testContext(t)
	userID := randomUUID(t)
	t.Cleanup(func() {
		deleteUserData(t, pool, userID)
	})

	incomeCategory := createCategory(t, ctx, repo, userID, "Salary", transaction.TypeIncome)
	expenseCategory := createCategory(t, ctx, repo, userID, "Food", transaction.TypeExpense)

	income := createTransaction(t, ctx, repo, transaction.CreateTransactionParams{
		UserID:      userID,
		CategoryID:  incomeCategory.ID,
		Type:        transaction.TypeIncome,
		Amount:      100_000,
		Currency:    "RUB",
		Description: "salary",
		OccurredAt:  time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC),
	})
	food := createTransaction(t, ctx, repo, transaction.CreateTransactionParams{
		UserID:      userID,
		CategoryID:  expenseCategory.ID,
		Type:        transaction.TypeExpense,
		Amount:      15_000,
		Currency:    "RUB",
		Description: "groceries",
		OccurredAt:  time.Date(2026, 1, 6, 10, 0, 0, 0, time.UTC),
	})
	coffee := createTransaction(t, ctx, repo, transaction.CreateTransactionParams{
		UserID:      userID,
		CategoryID:  expenseCategory.ID,
		Type:        transaction.TypeExpense,
		Amount:      500,
		Currency:    "USD",
		Description: "coffee",
		OccurredAt:  time.Date(2026, 1, 7, 10, 0, 0, 0, time.UTC),
	})

	if income.ID == "" || food.ID == "" || coffee.ID == "" {
		t.Fatal("expected generated transaction ids")
	}

	transactionType := transaction.TypeExpense
	history, err := repo.ListTransactions(ctx, transaction.ListTransactionsFilter{
		UserID: userID,
		Type:   &transactionType,
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}

	if history.TotalCount != 2 {
		t.Fatalf("expected total count 2, got %d", history.TotalCount)
	}

	if len(history.Transactions) != 1 || history.Transactions[0].ID != coffee.ID {
		t.Fatalf("expected latest expense transaction, got %+v", history.Transactions)
	}

	categoryID := expenseCategory.ID
	from := time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 6, 23, 59, 59, 0, time.UTC)
	filtered, err := repo.ListTransactions(ctx, transaction.ListTransactionsFilter{
		UserID:     userID,
		From:       &from,
		To:         &to,
		CategoryID: &categoryID,
	})
	if err != nil {
		t.Fatalf("list filtered transactions: %v", err)
	}

	if filtered.TotalCount != 1 || len(filtered.Transactions) != 1 || filtered.Transactions[0].ID != food.ID {
		t.Fatalf("expected food transaction, got %+v", filtered)
	}

	balances, err := repo.GetBalance(ctx, userID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	assertBalance(t, balances, "RUB", 100_000, 15_000, 85_000)
	assertBalance(t, balances, "USD", 0, 500, -500)
}

func TestRepositoryCreateTransactionRequiresMatchingCategoryType(t *testing.T) {
	repo, pool := newTestRepository(t)
	ctx := testContext(t)
	userID := randomUUID(t)
	t.Cleanup(func() {
		deleteUserData(t, pool, userID)
	})

	expenseCategory := createCategory(t, ctx, repo, userID, "Food", transaction.TypeExpense)

	_, err := repo.CreateTransaction(ctx, transaction.CreateTransactionParams{
		UserID:     userID,
		CategoryID: expenseCategory.ID,
		Type:       transaction.TypeIncome,
		Amount:     10_000,
		Currency:   "RUB",
		OccurredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, transaction.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRepositoryListTransactionsReturnsErrInvalidID(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := testContext(t)

	_, err := repo.ListTransactions(ctx, transaction.ListTransactionsFilter{
		UserID: "not-a-uuid",
	})
	if !errors.Is(err, transaction.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func newTestRepository(t *testing.T) (*Repository, *pgxpool.Pool) {
	t.Helper()

	dsn := os.Getenv("TRANSACTIONS_TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TRANSACTIONS_TEST_DATABASE_DSN is not set")
	}

	ctx := testContext(t)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	return New(pool), pool
}

func testContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	return ctx
}

func randomUUID(t *testing.T) string {
	t.Helper()

	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		t.Fatalf("generate uuid bytes: %v", err)
	}

	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80

	return hex.EncodeToString(bytes[0:4]) + "-" +
		hex.EncodeToString(bytes[4:6]) + "-" +
		hex.EncodeToString(bytes[6:8]) + "-" +
		hex.EncodeToString(bytes[8:10]) + "-" +
		hex.EncodeToString(bytes[10:16])
}

func createCategory(
	t *testing.T,
	ctx context.Context,
	repo *Repository,
	userID string,
	name string,
	categoryType transaction.Type,
) transaction.Category {
	t.Helper()

	category, err := repo.CreateCategory(ctx, transaction.CreateCategoryParams{
		UserID: userID,
		Name:   name,
		Type:   categoryType,
	})
	if err != nil {
		t.Fatalf("create category %q: %v", name, err)
	}

	return category
}

func createTransaction(
	t *testing.T,
	ctx context.Context,
	repo *Repository,
	params transaction.CreateTransactionParams,
) transaction.Transaction {
	t.Helper()

	item, err := repo.CreateTransaction(ctx, params)
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	return item
}

func assertBalance(
	t *testing.T,
	balances []transaction.CurrencyBalance,
	currency string,
	incomeAmount int64,
	expenseAmount int64,
	balanceAmount int64,
) {
	t.Helper()

	for _, balance := range balances {
		if balance.Currency != currency {
			continue
		}

		if balance.IncomeAmount != incomeAmount ||
			balance.ExpenseAmount != expenseAmount ||
			balance.BalanceAmount != balanceAmount {
			t.Fatalf("unexpected %s balance: %+v", currency, balance)
		}

		return
	}

	t.Fatalf("expected %s balance in %+v", currency, balances)
}

func deleteUserData(t *testing.T, pool *pgxpool.Pool, userID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1::uuid", userID); err != nil {
		t.Fatalf("delete test transactions: %v", err)
	}

	if _, err := pool.Exec(ctx, "DELETE FROM categories WHERE user_id = $1::uuid", userID); err != nil {
		t.Fatalf("delete test categories: %v", err)
	}
}
