package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	analyticsv1 "github.com/DekuMidBak/gofintracker/gen/go/analytics/v1"
	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
	"google.golang.org/grpc"
)

func TestGetMonthlyAnalytics(t *testing.T) {
	analytics := &fakeAnalyticsClient{
		monthlyResponse: &analyticsv1.GetMonthlySummaryResponse{
			Summaries: []*analyticsv1.MonthlyCurrencySummary{
				{
					Currency:      "RUB",
					IncomeAmount:  100_000,
					ExpenseAmount: 25_000,
					BalanceAmount: 75_000,
				},
			},
		},
	}
	router := newTestRouterWithClients(Clients{
		Users: &fakeUserClient{
			validateResponse: &userv1.ValidateTokenResponse{UserId: "user-1"},
		},
		Analytics: analytics,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/monthly?year=2026&month=5", nil)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if analytics.monthlyRequest.GetUserId() != "user-1" {
		t.Fatalf("expected user id from token, got %q", analytics.monthlyRequest.GetUserId())
	}

	if analytics.monthlyRequest.GetYear() != 2026 || analytics.monthlyRequest.GetMonth() != 5 {
		t.Fatalf("unexpected period: year=%d month=%d", analytics.monthlyRequest.GetYear(), analytics.monthlyRequest.GetMonth())
	}

	var body struct {
		Summaries []monthlySummaryResponse `json:"summaries"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Summaries) != 1 || body.Summaries[0].BalanceAmount != 75_000 {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestGetCategoryAnalytics(t *testing.T) {
	analytics := &fakeAnalyticsClient{
		categoryResponse: &analyticsv1.GetCategoryStatsResponse{
			Stats: []*analyticsv1.CategoryStat{
				{
					CategoryId: "category-1",
					Currency:   "RUB",
					Type:       analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_EXPENSE,
					Amount:     25_000,
				},
			},
		},
	}
	router := newTestRouterWithClients(Clients{
		Users: &fakeUserClient{
			validateResponse: &userv1.ValidateTokenResponse{UserId: "user-1"},
		},
		Analytics: analytics,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/categories?year=2026&month=5&type=expense", nil)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if analytics.categoryRequest.GetUserId() != "user-1" {
		t.Fatalf("expected user id from token, got %q", analytics.categoryRequest.GetUserId())
	}

	if analytics.categoryRequest.Type == nil ||
		analytics.categoryRequest.GetType() != analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_EXPENSE {
		t.Fatalf("expected expense type filter, got %+v", analytics.categoryRequest.Type)
	}

	var body struct {
		Stats []categoryStatResponse `json:"stats"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Stats) != 1 || body.Stats[0].CategoryID != "category-1" || body.Stats[0].Type != "expense" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestAnalyticsRejectsInvalidPeriod(t *testing.T) {
	router := newTestRouterWithClients(Clients{
		Users: &fakeUserClient{
			validateResponse: &userv1.ValidateTokenResponse{UserId: "user-1"},
		},
		Analytics: &fakeAnalyticsClient{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/monthly?year=bad&month=5", nil)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestAnalyticsRequireAuth(t *testing.T) {
	router := newTestRouterWithClients(Clients{
		Users:     &fakeUserClient{},
		Analytics: &fakeAnalyticsClient{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/monthly?year=2026&month=5", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

type fakeAnalyticsClient struct {
	monthlyRequest  *analyticsv1.GetMonthlySummaryRequest
	monthlyResponse *analyticsv1.GetMonthlySummaryResponse
	monthlyErr      error

	categoryRequest  *analyticsv1.GetCategoryStatsRequest
	categoryResponse *analyticsv1.GetCategoryStatsResponse
	categoryErr      error
}

func (c *fakeAnalyticsClient) GetMonthlySummary(
	_ context.Context,
	in *analyticsv1.GetMonthlySummaryRequest,
	_ ...grpc.CallOption,
) (*analyticsv1.GetMonthlySummaryResponse, error) {
	c.monthlyRequest = in
	if c.monthlyErr != nil {
		return nil, c.monthlyErr
	}

	return c.monthlyResponse, nil
}

func (c *fakeAnalyticsClient) GetCategoryStats(
	_ context.Context,
	in *analyticsv1.GetCategoryStatsRequest,
	_ ...grpc.CallOption,
) (*analyticsv1.GetCategoryStatsResponse, error) {
	c.categoryRequest = in
	if c.categoryErr != nil {
		return nil, c.categoryErr
	}

	return c.categoryResponse, nil
}
