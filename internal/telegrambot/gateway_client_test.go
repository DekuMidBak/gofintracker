package telegrambot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGatewayClientRegister(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/auth/register" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if body["email"] != "user@example.com" || body["password"] != "secret" {
			t.Fatalf("unexpected request body: %+v", body)
		}

		writeTestJSON(w, http.StatusCreated, AuthResult{
			UserID:      "user-1",
			AccessToken: "token-1",
		})
	}))
	defer server.Close()

	client := NewGatewayClient(server.URL, server.Client())
	result, err := client.Register(context.Background(), "user@example.com", "secret")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if result.UserID != "user-1" || result.AccessToken != "token-1" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestGatewayClientCreateCategoryUsesBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token-1" {
			t.Fatalf("unexpected authorization header: %q", r.Header.Get("Authorization"))
		}

		writeTestJSON(w, http.StatusCreated, Category{
			ID:   "category-1",
			Name: "Food",
			Type: "expense",
		})
	}))
	defer server.Close()

	client := NewGatewayClient(server.URL, server.Client())
	category, err := client.CreateCategory(context.Background(), "token-1", "Food", "expense")
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	if category.ID != "category-1" || category.Type != "expense" {
		t.Fatalf("unexpected category: %+v", category)
	}
}

func TestGatewayClientCreateTransaction(t *testing.T) {
	occurredAt := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if body["category_id"] != "category-1" || body["type"] != "expense" {
			t.Fatalf("unexpected transaction body: %+v", body)
		}

		if body["occurred_at"] != "2026-05-31T12:00:00Z" {
			t.Fatalf("unexpected occurred_at: %v", body["occurred_at"])
		}

		writeTestJSON(w, http.StatusCreated, Transaction{
			ID:         "transaction-1",
			CategoryID: "category-1",
			Type:       "expense",
			Amount:     1500,
			Currency:   "RUB",
			OccurredAt: occurredAt,
		})
	}))
	defer server.Close()

	client := NewGatewayClient(server.URL, server.Client())
	transaction, err := client.CreateTransaction(context.Background(), "token-1", CreateTransactionParams{
		CategoryID:  "category-1",
		Type:        "expense",
		Amount:      1500,
		Currency:    "RUB",
		Description: "Lunch",
		OccurredAt:  occurredAt,
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	if transaction.ID != "transaction-1" || transaction.Amount != 1500 {
		t.Fatalf("unexpected transaction: %+v", transaction)
	}
}

func TestGatewayClientGetCategoryStatsBuildsQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("year") != "2026" || query.Get("month") != "5" || query.Get("type") != "expense" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}

		writeTestJSON(w, http.StatusOK, map[string]any{
			"stats": []CategoryStat{
				{
					CategoryID: "category-1",
					Currency:   "RUB",
					Type:       "expense",
					Amount:     1500,
				},
			},
		})
	}))
	defer server.Close()

	client := NewGatewayClient(server.URL, server.Client())
	stats, err := client.GetCategoryStats(context.Background(), "token-1", 2026, 5, "expense")
	if err != nil {
		t.Fatalf("get category stats: %v", err)
	}

	if len(stats) != 1 || stats[0].Amount != 1500 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestGatewayClientReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "missing token",
		})
	}))
	defer server.Close()

	client := NewGatewayClient(server.URL, server.Client())
	_, err := client.GetBalance(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusUnauthorized || apiErr.Message != "missing token" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func writeTestJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
