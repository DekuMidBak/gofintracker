package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/DekuMidBak/gofintracker/internal/analytics"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryApplyTransactionCreatedUpdatesAggregates(t *testing.T) {
	repo, pool := newTestRepository(t)
	ctx := testContext(t)
	userID := randomUUID(t)
	categoryID := randomUUID(t)
	t.Cleanup(func() {
		deleteUserData(t, pool, userID)
	})

	income := newEvent(t, userID, categoryID, analytics.TransactionTypeIncome, 100_000, "RUB")
	applied, err := repo.ApplyTransactionCreated(ctx, income)
	if err != nil {
		t.Fatalf("apply income event: %v", err)
	}

	if !applied {
		t.Fatal("expected first event to be applied")
	}

	expense := newEvent(t, userID, categoryID, analytics.TransactionTypeExpense, 15_000, "RUB")
	applied, err = repo.ApplyTransactionCreated(ctx, expense)
	if err != nil {
		t.Fatalf("apply expense event: %v", err)
	}

	if !applied {
		t.Fatal("expected second event to be applied")
	}

	summaries, err := repo.GetMonthlySummary(ctx, userID, 2026, 1)
	if err != nil {
		t.Fatalf("get monthly summary: %v", err)
	}

	if len(summaries) != 1 {
		t.Fatalf("expected one summary, got %+v", summaries)
	}

	if summaries[0].IncomeAmount != 100_000 ||
		summaries[0].ExpenseAmount != 15_000 ||
		summaries[0].BalanceAmount != 85_000 {
		t.Fatalf("unexpected summary: %+v", summaries[0])
	}

	stats, err := repo.GetCategoryStats(ctx, analytics.CategoryStatsFilter{
		UserID: userID,
		Year:   2026,
		Month:  1,
	})
	if err != nil {
		t.Fatalf("get category stats: %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("expected two category stats, got %+v", stats)
	}
}

func TestRepositoryApplyTransactionCreatedIsIdempotent(t *testing.T) {
	repo, pool := newTestRepository(t)
	ctx := testContext(t)
	userID := randomUUID(t)
	categoryID := randomUUID(t)
	t.Cleanup(func() {
		deleteUserData(t, pool, userID)
	})

	event := newEvent(t, userID, categoryID, analytics.TransactionTypeExpense, 1500, "RUB")
	applied, err := repo.ApplyTransactionCreated(ctx, event)
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}

	if !applied {
		t.Fatal("expected first event to be applied")
	}

	applied, err = repo.ApplyTransactionCreated(ctx, event)
	if err != nil {
		t.Fatalf("apply duplicate event: %v", err)
	}

	if applied {
		t.Fatal("expected duplicate event not to be applied")
	}

	summaries, err := repo.GetMonthlySummary(ctx, userID, 2026, 1)
	if err != nil {
		t.Fatalf("get monthly summary: %v", err)
	}

	if len(summaries) != 1 || summaries[0].ExpenseAmount != 1500 {
		t.Fatalf("expected event to be counted once, got %+v", summaries)
	}
}

func TestRepositoryGetCategoryStatsFiltersByType(t *testing.T) {
	repo, pool := newTestRepository(t)
	ctx := testContext(t)
	userID := randomUUID(t)
	categoryID := randomUUID(t)
	t.Cleanup(func() {
		deleteUserData(t, pool, userID)
	})

	if _, err := repo.ApplyTransactionCreated(ctx, newEvent(t, userID, categoryID, analytics.TransactionTypeIncome, 1000, "RUB")); err != nil {
		t.Fatalf("apply income event: %v", err)
	}

	if _, err := repo.ApplyTransactionCreated(ctx, newEvent(t, userID, categoryID, analytics.TransactionTypeExpense, 500, "RUB")); err != nil {
		t.Fatalf("apply expense event: %v", err)
	}

	statType := analytics.TransactionTypeExpense
	stats, err := repo.GetCategoryStats(ctx, analytics.CategoryStatsFilter{
		UserID: userID,
		Year:   2026,
		Month:  1,
		Type:   &statType,
	})
	if err != nil {
		t.Fatalf("get category stats: %v", err)
	}

	if len(stats) != 1 || stats[0].Type != analytics.TransactionTypeExpense || stats[0].Amount != 500 {
		t.Fatalf("expected only expense stat, got %+v", stats)
	}
}

func TestRepositoryApplyTransactionCreatedReturnsErrInvalidID(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := testContext(t)

	event := newEvent(t, randomUUID(t), randomUUID(t), analytics.TransactionTypeExpense, 1500, "RUB")
	event.EventID = "not-a-uuid"

	_, err := repo.ApplyTransactionCreated(ctx, event)
	if !errors.Is(err, analytics.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func newTestRepository(t *testing.T) (*Repository, *pgxpool.Pool) {
	t.Helper()

	dsn := os.Getenv("ANALYTICS_TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("ANALYTICS_TEST_DATABASE_DSN is not set")
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

func newEvent(
	t *testing.T,
	userID string,
	categoryID string,
	transactionType analytics.TransactionType,
	amount int64,
	currency string,
) analytics.TransactionCreated {
	t.Helper()

	return analytics.TransactionCreated{
		EventID:       randomUUID(t),
		UserID:        userID,
		TransactionID: randomUUID(t),
		Type:          transactionType,
		Amount:        amount,
		Currency:      currency,
		CategoryID:    categoryID,
		OccurredAt:    time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		CreatedAt:     time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
	}
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

func deleteUserData(t *testing.T, pool *pgxpool.Pool, userID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx, "DELETE FROM category_aggregates WHERE user_id = $1::uuid", userID); err != nil {
		t.Fatalf("delete category aggregates: %v", err)
	}

	if _, err := pool.Exec(ctx, "DELETE FROM monthly_aggregates WHERE user_id = $1::uuid", userID); err != nil {
		t.Fatalf("delete monthly aggregates: %v", err)
	}
}
