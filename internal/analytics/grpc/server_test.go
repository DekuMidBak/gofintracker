package grpc

import (
	"context"
	"errors"
	"testing"

	analyticsv1 "github.com/DekuMidBak/gofintracker/gen/go/analytics/v1"
	"github.com/DekuMidBak/gofintracker/internal/analytics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServerGetMonthlySummary(t *testing.T) {
	service := &fakeService{
		monthlySummaries: []analytics.MonthlySummary{
			{
				Currency:      "RUB",
				IncomeAmount:  100_000,
				ExpenseAmount: 15_000,
				BalanceAmount: 85_000,
			},
		},
	}
	server := NewServer(service)

	resp, err := server.GetMonthlySummary(context.Background(), &analyticsv1.GetMonthlySummaryRequest{
		UserId: "user-1",
		Year:   2026,
		Month:  1,
	})
	if err != nil {
		t.Fatalf("get monthly summary: %v", err)
	}

	if service.monthlyUserID != "user-1" || service.monthlyYear != 2026 || service.monthlyMonth != 1 {
		t.Fatalf("expected request to be forwarded")
	}

	if len(resp.GetSummaries()) != 1 || resp.GetSummaries()[0].GetBalanceAmount() != 85_000 {
		t.Fatalf("expected monthly summary, got %+v", resp.GetSummaries())
	}
}

func TestServerGetCategoryStatsForwardsOptionalType(t *testing.T) {
	statType := analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_EXPENSE
	service := &fakeService{
		categoryStats: []analytics.CategoryStat{
			{
				CategoryID: "category-1",
				Currency:   "RUB",
				Type:       analytics.TransactionTypeExpense,
				Amount:     15_000,
			},
		},
	}
	server := NewServer(service)

	resp, err := server.GetCategoryStats(context.Background(), &analyticsv1.GetCategoryStatsRequest{
		UserId: "user-1",
		Year:   2026,
		Month:  1,
		Type:   &statType,
	})
	if err != nil {
		t.Fatalf("get category stats: %v", err)
	}

	filter := service.categoryFilter
	if filter.Type == nil || *filter.Type != analytics.TransactionTypeExpense {
		t.Fatalf("expected expense type filter, got %+v", filter.Type)
	}

	if len(resp.GetStats()) != 1 || resp.GetStats()[0].GetType() != statType {
		t.Fatalf("expected category stat response, got %+v", resp.GetStats())
	}
}

func TestServerMapsValidationErrorToInvalidArgument(t *testing.T) {
	server := NewServer(&fakeService{
		monthlyErr: analytics.ErrInvalidPeriod,
	})

	_, err := server.GetMonthlySummary(context.Background(), &analyticsv1.GetMonthlySummaryRequest{
		UserId: "user-1",
		Year:   2026,
		Month:  13,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %s", status.Code(err))
	}
}

func TestServerMapsUnknownErrorToInternal(t *testing.T) {
	server := NewServer(&fakeService{
		monthlyErr: errors.New("boom"),
	})

	_, err := server.GetMonthlySummary(context.Background(), &analyticsv1.GetMonthlySummaryRequest{})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal, got %s", status.Code(err))
	}
}

type fakeService struct {
	monthlyUserID    string
	monthlyYear      int
	monthlyMonth     int
	monthlySummaries []analytics.MonthlySummary
	monthlyErr       error

	categoryFilter analytics.CategoryStatsFilter
	categoryStats  []analytics.CategoryStat
	categoryErr    error
}

func (s *fakeService) GetMonthlySummary(
	_ context.Context,
	userID string,
	year int,
	month int,
) ([]analytics.MonthlySummary, error) {
	s.monthlyUserID = userID
	s.monthlyYear = year
	s.monthlyMonth = month
	if s.monthlyErr != nil {
		return nil, s.monthlyErr
	}

	return s.monthlySummaries, nil
}

func (s *fakeService) GetCategoryStats(
	_ context.Context,
	filter analytics.CategoryStatsFilter,
) ([]analytics.CategoryStat, error) {
	s.categoryFilter = filter
	if s.categoryErr != nil {
		return nil, s.categoryErr
	}

	return s.categoryStats, nil
}
