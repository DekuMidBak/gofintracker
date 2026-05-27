package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	transactionv1 "github.com/DekuMidBak/gofintracker/gen/go/transaction/v1"
	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCreateTransaction(t *testing.T) {
	occurredAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	transactions := &fakeTransactionClient{
		createTransactionResponse: &transactionv1.CreateTransactionResponse{
			Transaction: &transactionv1.Transaction{
				Id:          "transaction-1",
				UserId:      "user-1",
				CategoryId:  "category-1",
				Type:        transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE,
				Amount:      1500,
				Currency:    "RUB",
				Description: "Lunch",
				OccurredAt:  timestamppb.New(occurredAt),
				CreatedAt:   timestamppb.New(occurredAt),
			},
		},
	}
	router := newTestRouterWithClients(Clients{
		Users: &fakeUserClient{
			validateResponse: &userv1.ValidateTokenResponse{UserId: "user-1"},
		},
		Transactions: transactions,
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/transactions",
		strings.NewReader(`{"category_id":"category-1","type":"expense","amount":1500,"currency":"RUB","description":"Lunch","occurred_at":"2026-01-02T03:04:05Z"}`),
	)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	if transactions.createTransactionRequest.GetUserId() != "user-1" {
		t.Fatalf("expected user id from token, got %q", transactions.createTransactionRequest.GetUserId())
	}

	if transactions.createTransactionRequest.GetType() != transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE {
		t.Fatalf("expected expense type, got %s", transactions.createTransactionRequest.GetType())
	}

	if transactions.createTransactionRequest.GetOccurredAt().AsTime() != occurredAt {
		t.Fatalf("expected occurred_at to be forwarded")
	}

	var body transactionResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.ID != "transaction-1" || body.Type != "expense" || body.Amount != 1500 {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestCreateTransactionRejectsInvalidOccurredAt(t *testing.T) {
	router := newTestRouterWithClients(Clients{
		Users: &fakeUserClient{
			validateResponse: &userv1.ValidateTokenResponse{UserId: "user-1"},
		},
		Transactions: &fakeTransactionClient{},
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/transactions",
		strings.NewReader(`{"category_id":"category-1","type":"expense","amount":1500,"currency":"RUB","occurred_at":"bad"}`),
	)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestListTransactions(t *testing.T) {
	occurredAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	transactions := &fakeTransactionClient{
		listTransactionsResponse: &transactionv1.ListTransactionsResponse{
			Transactions: []*transactionv1.Transaction{
				{
					Id:         "transaction-1",
					UserId:     "user-1",
					CategoryId: "category-1",
					Type:       transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE,
					Amount:     1500,
					Currency:   "RUB",
					OccurredAt: timestamppb.New(occurredAt),
					CreatedAt:  timestamppb.New(occurredAt),
				},
			},
			TotalCount: 1,
		},
	}
	router := newTestRouterWithClients(Clients{
		Users: &fakeUserClient{
			validateResponse: &userv1.ValidateTokenResponse{UserId: "user-1"},
		},
		Transactions: transactions,
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/transactions?from=2026-01-01T00:00:00Z&to=2026-01-31T00:00:00Z&category_id=category-1&type=expense&limit=20&offset=5",
		nil,
	)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	request := transactions.listTransactionsRequest
	if request.GetUserId() != "user-1" {
		t.Fatalf("expected user id from token, got %q", request.GetUserId())
	}

	if request.CategoryId == nil || request.GetCategoryId() != "category-1" {
		t.Fatalf("expected category filter, got %+v", request.CategoryId)
	}

	if request.Type == nil || request.GetType() != transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE {
		t.Fatalf("expected expense type filter, got %+v", request.Type)
	}

	if request.GetLimit() != 20 || request.GetOffset() != 5 {
		t.Fatalf("expected pagination, got limit=%d offset=%d", request.GetLimit(), request.GetOffset())
	}

	var body struct {
		Transactions []transactionResponse `json:"transactions"`
		TotalCount   int32                 `json:"total_count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Transactions) != 1 || body.TotalCount != 1 {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestListTransactionsRejectsLimitOverflow(t *testing.T) {
	router := newTestRouterWithClients(Clients{
		Users: &fakeUserClient{
			validateResponse: &userv1.ValidateTokenResponse{UserId: "user-1"},
		},
		Transactions: &fakeTransactionClient{},
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/transactions?limit=2147483648",
		nil,
	)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetBalance(t *testing.T) {
	transactions := &fakeTransactionClient{
		getBalanceResponse: &transactionv1.GetBalanceResponse{
			Balances: []*transactionv1.CurrencyBalance{
				{
					Currency:      "RUB",
					IncomeAmount:  100_000,
					ExpenseAmount: 15_000,
					BalanceAmount: 85_000,
				},
			},
		},
	}
	router := newTestRouterWithClients(Clients{
		Users: &fakeUserClient{
			validateResponse: &userv1.ValidateTokenResponse{UserId: "user-1"},
		},
		Transactions: transactions,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/balance", nil)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if transactions.getBalanceRequest.GetUserId() != "user-1" {
		t.Fatalf("expected user id from token, got %q", transactions.getBalanceRequest.GetUserId())
	}

	var body struct {
		Balances []balanceResponse `json:"balances"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Balances) != 1 || body.Balances[0].BalanceAmount != 85_000 {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestTransactionsRequireAuth(t *testing.T) {
	router := newTestRouterWithClients(Clients{
		Users:        &fakeUserClient{},
		Transactions: &fakeTransactionClient{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}
