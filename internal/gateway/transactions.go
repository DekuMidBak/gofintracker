package gateway

import (
	"net/http"
	"strconv"
	"time"

	transactionv1 "github.com/DekuMidBak/gofintracker/gen/go/transaction/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type transactionRequest struct {
	CategoryID  string `json:"category_id"`
	Type        string `json:"type"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	OccurredAt  string `json:"occurred_at"`
}

type transactionResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	CategoryID  string    `json:"category_id"`
	Type        string    `json:"type"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	Description string    `json:"description"`
	OccurredAt  time.Time `json:"occurred_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type balanceResponse struct {
	Currency      string `json:"currency"`
	IncomeAmount  int64  `json:"income_amount"`
	ExpenseAmount int64  `json:"expense_amount"`
	BalanceAmount int64  `json:"balance_amount"`
}

func (h handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	if h.clients.Transactions == nil {
		writeError(w, http.StatusInternalServerError, "service dependency is not configured")
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	var req transactionRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	transactionType, ok := transactionTypeFromString(req.Type)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid transaction type")
		return
	}

	occurredAt, err := parseRequiredTime(req.OccurredAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid occurred_at")
		return
	}

	resp, err := h.clients.Transactions.CreateTransaction(r.Context(), &transactionv1.CreateTransactionRequest{
		UserId:      userID,
		CategoryId:  req.CategoryID,
		Type:        transactionType,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		OccurredAt:  timestamppb.New(occurredAt),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toTransactionResponse(resp.GetTransaction()))
}

func (h handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	if h.clients.Transactions == nil {
		writeError(w, http.StatusInternalServerError, "service dependency is not configured")
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	query := r.URL.Query()
	req := &transactionv1.ListTransactionsRequest{
		UserId: userID,
	}

	from, err := parseOptionalTime(query.Get("from"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid from")
		return
	}
	if !from.IsZero() {
		req.From = timestamppb.New(from)
	}

	to, err := parseOptionalTime(query.Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid to")
		return
	}
	if !to.IsZero() {
		req.To = timestamppb.New(to)
	}

	if categoryID := query.Get("category_id"); categoryID != "" {
		req.CategoryId = &categoryID
	}

	if rawType := query.Get("type"); rawType != "" {
		transactionType, ok := transactionTypeFromString(rawType)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid transaction type")
			return
		}

		req.Type = &transactionType
	}

	limit, err := parseOptionalInt32(query.Get("limit"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid limit")
		return
	}
	req.Limit = limit

	offset, err := parseOptionalInt32(query.Get("offset"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offset")
		return
	}
	req.Offset = offset

	resp, err := h.clients.Transactions.ListTransactions(r.Context(), req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	transactions := make([]transactionResponse, 0, len(resp.GetTransactions()))
	for _, item := range resp.GetTransactions() {
		transactions = append(transactions, toTransactionResponse(item))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"transactions": transactions,
		"total_count":  resp.GetTotalCount(),
	})
}

func (h handler) getBalance(w http.ResponseWriter, r *http.Request) {
	if h.clients.Transactions == nil {
		writeError(w, http.StatusInternalServerError, "service dependency is not configured")
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	resp, err := h.clients.Transactions.GetBalance(r.Context(), &transactionv1.GetBalanceRequest{
		UserId: userID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	balances := make([]balanceResponse, 0, len(resp.GetBalances()))
	for _, balance := range resp.GetBalances() {
		balances = append(balances, balanceResponse{
			Currency:      balance.GetCurrency(),
			IncomeAmount:  balance.GetIncomeAmount(),
			ExpenseAmount: balance.GetExpenseAmount(),
			BalanceAmount: balance.GetBalanceAmount(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"balances": balances,
	})
}

func parseRequiredTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, strconv.ErrSyntax
	}

	return time.Parse(time.RFC3339, value)
}

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	return time.Parse(time.RFC3339, value)
}

func parseOptionalInt32(value string) (int32, error) {
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(parsed), nil
}

func toTransactionResponse(item *transactionv1.Transaction) transactionResponse {
	var occurredAt time.Time
	if item.GetOccurredAt() != nil {
		occurredAt = item.GetOccurredAt().AsTime()
	}

	var createdAt time.Time
	if item.GetCreatedAt() != nil {
		createdAt = item.GetCreatedAt().AsTime()
	}

	return transactionResponse{
		ID:          item.GetId(),
		UserID:      item.GetUserId(),
		CategoryID:  item.GetCategoryId(),
		Type:        transactionTypeToString(item.GetType()),
		Amount:      item.GetAmount(),
		Currency:    item.GetCurrency(),
		Description: item.GetDescription(),
		OccurredAt:  occurredAt,
		CreatedAt:   createdAt,
	}
}
