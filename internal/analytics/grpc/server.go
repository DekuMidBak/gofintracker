package grpc

import (
	"context"
	"errors"

	analyticsv1 "github.com/DekuMidBak/gofintracker/gen/go/analytics/v1"
	"github.com/DekuMidBak/gofintracker/internal/analytics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service interface {
	GetMonthlySummary(ctx context.Context, userID string, year int, month int) ([]analytics.MonthlySummary, error)
	GetCategoryStats(ctx context.Context, filter analytics.CategoryStatsFilter) ([]analytics.CategoryStat, error)
}

type Server struct {
	analyticsv1.UnimplementedAnalyticsServiceServer

	service Service
}

var _ analyticsv1.AnalyticsServiceServer = (*Server)(nil)

func NewServer(service Service) *Server {
	return &Server{service: service}
}

func (s *Server) GetMonthlySummary(
	ctx context.Context,
	req *analyticsv1.GetMonthlySummaryRequest,
) (*analyticsv1.GetMonthlySummaryResponse, error) {
	summaries, err := s.service.GetMonthlySummary(
		ctx,
		req.GetUserId(),
		int(req.GetYear()),
		int(req.GetMonth()),
	)
	if err != nil {
		return nil, mapError(err)
	}

	return &analyticsv1.GetMonthlySummaryResponse{
		Summaries: toProtoMonthlySummaries(summaries),
	}, nil
}

func (s *Server) GetCategoryStats(
	ctx context.Context,
	req *analyticsv1.GetCategoryStatsRequest,
) (*analyticsv1.GetCategoryStatsResponse, error) {
	var statType *analytics.TransactionType
	if req.Type != nil {
		converted := fromProtoType(req.GetType())
		statType = &converted
	}

	stats, err := s.service.GetCategoryStats(ctx, analytics.CategoryStatsFilter{
		UserID: req.GetUserId(),
		Year:   int(req.GetYear()),
		Month:  int(req.GetMonth()),
		Type:   statType,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &analyticsv1.GetCategoryStatsResponse{
		Stats: toProtoCategoryStats(stats),
	}, nil
}

func fromProtoType(protoType analyticsv1.AnalyticsTransactionType) analytics.TransactionType {
	switch protoType {
	case analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_INCOME:
		return analytics.TransactionTypeIncome
	case analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_EXPENSE:
		return analytics.TransactionTypeExpense
	default:
		return analytics.TransactionType("")
	}
}

func toProtoType(transactionType analytics.TransactionType) analyticsv1.AnalyticsTransactionType {
	switch transactionType {
	case analytics.TransactionTypeIncome:
		return analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_INCOME
	case analytics.TransactionTypeExpense:
		return analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_EXPENSE
	default:
		return analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_UNSPECIFIED
	}
}

func toProtoMonthlySummaries(summaries []analytics.MonthlySummary) []*analyticsv1.MonthlyCurrencySummary {
	result := make([]*analyticsv1.MonthlyCurrencySummary, 0, len(summaries))
	for _, summary := range summaries {
		result = append(result, &analyticsv1.MonthlyCurrencySummary{
			Currency:      summary.Currency,
			IncomeAmount:  summary.IncomeAmount,
			ExpenseAmount: summary.ExpenseAmount,
			BalanceAmount: summary.BalanceAmount,
		})
	}

	return result
}

func toProtoCategoryStats(stats []analytics.CategoryStat) []*analyticsv1.CategoryStat {
	result := make([]*analyticsv1.CategoryStat, 0, len(stats))
	for _, stat := range stats {
		result = append(result, &analyticsv1.CategoryStat{
			CategoryId: stat.CategoryID,
			Currency:   stat.Currency,
			Type:       toProtoType(stat.Type),
			Amount:     stat.Amount,
		})
	}

	return result
}

func mapError(err error) error {
	switch {
	case errors.Is(err, analytics.ErrInvalidID),
		errors.Is(err, analytics.ErrInvalidPeriod),
		errors.Is(err, analytics.ErrInvalidType),
		errors.Is(err, analytics.ErrInvalidAmount),
		errors.Is(err, analytics.ErrInvalidCurrency):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
