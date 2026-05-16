package analytics

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceProcessTransactionCreatedNormalizesAndAppliesEvent(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	applied, err := service.ProcessTransactionCreated(context.Background(), TransactionCreated{
		EventID:       " event-1 ",
		UserID:        " user-1 ",
		TransactionID: " transaction-1 ",
		Type:          TransactionTypeExpense,
		Amount:        1500,
		Currency:      " rub ",
		CategoryID:    " category-1 ",
		OccurredAt:    time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("UTC+3", 3*60*60)),
	})
	if err != nil {
		t.Fatalf("process event: %v", err)
	}

	if !applied {
		t.Fatal("expected event to be applied")
	}

	event := repository.appliedEvent
	if event.EventID != "event-1" {
		t.Fatalf("expected trimmed event id, got %q", event.EventID)
	}

	if event.Currency != "RUB" {
		t.Fatalf("expected normalized currency, got %q", event.Currency)
	}

	if event.OccurredAt.Location() != time.UTC {
		t.Fatalf("expected occurred_at in UTC, got %s", event.OccurredAt.Location())
	}

	if event.CreatedAt.IsZero() {
		t.Fatal("expected created_at fallback")
	}
}

func TestServiceProcessTransactionCreatedRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	valid := TransactionCreated{
		EventID:       "event-1",
		UserID:        "user-1",
		TransactionID: "transaction-1",
		Type:          TransactionTypeExpense,
		Amount:        1500,
		Currency:      "RUB",
		CategoryID:    "category-1",
		OccurredAt:    time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		CreatedAt:     time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
	}

	tests := []struct {
		name   string
		mutate func(*TransactionCreated)
		want   error
	}{
		{
			name: "empty id",
			mutate: func(event *TransactionCreated) {
				event.EventID = ""
			},
			want: ErrInvalidID,
		},
		{
			name: "invalid type",
			mutate: func(event *TransactionCreated) {
				event.Type = TransactionType("other")
			},
			want: ErrInvalidType,
		},
		{
			name: "non-positive amount",
			mutate: func(event *TransactionCreated) {
				event.Amount = 0
			},
			want: ErrInvalidAmount,
		},
		{
			name: "invalid currency",
			mutate: func(event *TransactionCreated) {
				event.Currency = "RU"
			},
			want: ErrInvalidCurrency,
		},
		{
			name: "zero occurred at",
			mutate: func(event *TransactionCreated) {
				event.OccurredAt = time.Time{}
			},
			want: ErrInvalidPeriod,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := valid
			tt.mutate(&event)

			_, err := service.ProcessTransactionCreated(context.Background(), event)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestServiceGetMonthlySummaryValidatesInput(t *testing.T) {
	repository := &fakeRepository{
		monthlySummaries: []MonthlySummary{
			{
				Currency:      "RUB",
				IncomeAmount:  100_000,
				ExpenseAmount: 15_000,
				BalanceAmount: 85_000,
			},
		},
	}
	service := NewService(repository)

	summaries, err := service.GetMonthlySummary(context.Background(), " user-1 ", 2026, 1)
	if err != nil {
		t.Fatalf("get monthly summary: %v", err)
	}

	if repository.monthlyUserID != "user-1" {
		t.Fatalf("expected trimmed user id, got %q", repository.monthlyUserID)
	}

	if len(summaries) != 1 || summaries[0].BalanceAmount != 85_000 {
		t.Fatalf("expected monthly summary, got %+v", summaries)
	}
}

func TestServiceGetMonthlySummaryRejectsInvalidPeriod(t *testing.T) {
	service := NewService(&fakeRepository{})

	_, err := service.GetMonthlySummary(context.Background(), "user-1", 2026, 13)
	if !errors.Is(err, ErrInvalidPeriod) {
		t.Fatalf("expected ErrInvalidPeriod, got %v", err)
	}
}

func TestServiceGetCategoryStatsValidatesAndForwardsFilter(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	statType := TransactionTypeIncome

	_, err := service.GetCategoryStats(context.Background(), CategoryStatsFilter{
		UserID: " user-1 ",
		Year:   2026,
		Month:  1,
		Type:   &statType,
	})
	if err != nil {
		t.Fatalf("get category stats: %v", err)
	}

	if repository.categoryFilter.UserID != "user-1" {
		t.Fatalf("expected trimmed user id, got %q", repository.categoryFilter.UserID)
	}

	if repository.categoryFilter.Type == nil || *repository.categoryFilter.Type != TransactionTypeIncome {
		t.Fatalf("expected income filter, got %+v", repository.categoryFilter.Type)
	}
}

func TestServiceGetCategoryStatsRejectsInvalidType(t *testing.T) {
	service := NewService(&fakeRepository{})
	statType := TransactionType("other")

	_, err := service.GetCategoryStats(context.Background(), CategoryStatsFilter{
		UserID: "user-1",
		Year:   2026,
		Month:  1,
		Type:   &statType,
	})
	if !errors.Is(err, ErrInvalidType) {
		t.Fatalf("expected ErrInvalidType, got %v", err)
	}
}

type fakeRepository struct {
	appliedEvent TransactionCreated
	applied      bool
	applyErr     error

	monthlyUserID    string
	monthlyYear      int
	monthlyMonth     int
	monthlySummaries []MonthlySummary
	monthlyErr       error

	categoryFilter CategoryStatsFilter
	categoryStats  []CategoryStat
	categoryErr    error
}

func (r *fakeRepository) ApplyTransactionCreated(_ context.Context, event TransactionCreated) (bool, error) {
	r.appliedEvent = event
	if r.applyErr != nil {
		return false, r.applyErr
	}

	return true, nil
}

func (r *fakeRepository) GetMonthlySummary(
	_ context.Context,
	userID string,
	year int,
	month int,
) ([]MonthlySummary, error) {
	r.monthlyUserID = userID
	r.monthlyYear = year
	r.monthlyMonth = month
	if r.monthlyErr != nil {
		return nil, r.monthlyErr
	}

	return r.monthlySummaries, nil
}

func (r *fakeRepository) GetCategoryStats(
	_ context.Context,
	filter CategoryStatsFilter,
) ([]CategoryStat, error) {
	r.categoryFilter = filter
	if r.categoryErr != nil {
		return nil, r.categoryErr
	}

	return r.categoryStats, nil
}
