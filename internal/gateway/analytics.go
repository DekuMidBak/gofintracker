package gateway

import (
	"net/http"
	"strconv"

	analyticsv1 "github.com/DekuMidBak/gofintracker/gen/go/analytics/v1"
)

type monthlySummaryResponse struct {
	Currency      string `json:"currency"`
	IncomeAmount  int64  `json:"income_amount"`
	ExpenseAmount int64  `json:"expense_amount"`
	BalanceAmount int64  `json:"balance_amount"`
}

type categoryStatResponse struct {
	CategoryID string `json:"category_id"`
	Currency   string `json:"currency"`
	Type       string `json:"type"`
	Amount     int64  `json:"amount"`
}

func (h handler) getMonthlyAnalytics(w http.ResponseWriter, r *http.Request) {
	if h.clients.Analytics == nil {
		writeError(w, http.StatusInternalServerError, "service dependency is not configured")
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	year, month, ok := parseAnalyticsPeriod(w, r)
	if !ok {
		return
	}

	resp, err := h.clients.Analytics.GetMonthlySummary(r.Context(), &analyticsv1.GetMonthlySummaryRequest{
		UserId: userID,
		Year:   int32(year),
		Month:  int32(month),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	summaries := make([]monthlySummaryResponse, 0, len(resp.GetSummaries()))
	for _, summary := range resp.GetSummaries() {
		summaries = append(summaries, monthlySummaryResponse{
			Currency:      summary.GetCurrency(),
			IncomeAmount:  summary.GetIncomeAmount(),
			ExpenseAmount: summary.GetExpenseAmount(),
			BalanceAmount: summary.GetBalanceAmount(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"summaries": summaries,
	})
}

func (h handler) getCategoryAnalytics(w http.ResponseWriter, r *http.Request) {
	if h.clients.Analytics == nil {
		writeError(w, http.StatusInternalServerError, "service dependency is not configured")
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	year, month, ok := parseAnalyticsPeriod(w, r)
	if !ok {
		return
	}

	req := &analyticsv1.GetCategoryStatsRequest{
		UserId: userID,
		Year:   int32(year),
		Month:  int32(month),
	}

	if rawType := r.URL.Query().Get("type"); rawType != "" {
		analyticsType, ok := analyticsTypeFromString(rawType)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid transaction type")
			return
		}

		req.Type = &analyticsType
	}

	resp, err := h.clients.Analytics.GetCategoryStats(r.Context(), req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	stats := make([]categoryStatResponse, 0, len(resp.GetStats()))
	for _, stat := range resp.GetStats() {
		stats = append(stats, categoryStatResponse{
			CategoryID: stat.GetCategoryId(),
			Currency:   stat.GetCurrency(),
			Type:       analyticsTypeToString(stat.GetType()),
			Amount:     stat.GetAmount(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"stats": stats,
	})
}

func parseAnalyticsPeriod(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid year")
		return 0, 0, false
	}

	month, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid month")
		return 0, 0, false
	}

	return year, month, true
}

func analyticsTypeFromString(value string) (analyticsv1.AnalyticsTransactionType, bool) {
	switch value {
	case "income":
		return analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_INCOME, true
	case "expense":
		return analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_EXPENSE, true
	default:
		return analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_UNSPECIFIED, false
	}
}

func analyticsTypeToString(value analyticsv1.AnalyticsTransactionType) string {
	switch value {
	case analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_INCOME:
		return "income"
	case analyticsv1.AnalyticsTransactionType_ANALYTICS_TRANSACTION_TYPE_EXPENSE:
		return "expense"
	default:
		return ""
	}
}
