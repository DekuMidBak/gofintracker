package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	transactionv1 "github.com/DekuMidBak/gofintracker/gen/go/transaction/v1"
	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCreateCategory(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	transactions := &fakeTransactionClient{
		createCategoryResponse: &transactionv1.CreateCategoryResponse{
			Category: &transactionv1.Category{
				Id:        "category-1",
				UserId:    "user-1",
				Name:      "Food",
				Type:      transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE,
				CreatedAt: timestamppb.New(createdAt),
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
		"/api/v1/categories",
		strings.NewReader(`{"name":"Food","type":"expense"}`),
	)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	if transactions.createCategoryRequest.GetUserId() != "user-1" {
		t.Fatalf("expected user id from token, got %q", transactions.createCategoryRequest.GetUserId())
	}

	if transactions.createCategoryRequest.GetType() != transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE {
		t.Fatalf("expected expense type, got %s", transactions.createCategoryRequest.GetType())
	}

	var body categoryResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.ID != "category-1" || body.Type != "expense" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestCreateCategoryRejectsInvalidType(t *testing.T) {
	router := newTestRouterWithClients(Clients{
		Users: &fakeUserClient{
			validateResponse: &userv1.ValidateTokenResponse{UserId: "user-1"},
		},
		Transactions: &fakeTransactionClient{},
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/categories",
		strings.NewReader(`{"name":"Food","type":"other"}`),
	)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestListCategories(t *testing.T) {
	transactions := &fakeTransactionClient{
		listCategoriesResponse: &transactionv1.ListCategoriesResponse{
			Categories: []*transactionv1.Category{
				{
					Id:     "category-1",
					UserId: "user-1",
					Name:   "Food",
					Type:   transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE,
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories?type=expense", nil)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if transactions.listCategoriesRequest.GetUserId() != "user-1" {
		t.Fatalf("expected user id from token, got %q", transactions.listCategoriesRequest.GetUserId())
	}

	if transactions.listCategoriesRequest.Type == nil ||
		transactions.listCategoriesRequest.GetType() != transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE {
		t.Fatalf("expected expense type filter, got %+v", transactions.listCategoriesRequest.Type)
	}

	var body struct {
		Categories []categoryResponse `json:"categories"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Categories) != 1 || body.Categories[0].ID != "category-1" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestCategoriesRequireAuth(t *testing.T) {
	router := newTestRouterWithClients(Clients{
		Users:        &fakeUserClient{},
		Transactions: &fakeTransactionClient{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

type fakeTransactionClient struct {
	createCategoryRequest  *transactionv1.CreateCategoryRequest
	createCategoryResponse *transactionv1.CreateCategoryResponse
	createCategoryErr      error

	listCategoriesRequest  *transactionv1.ListCategoriesRequest
	listCategoriesResponse *transactionv1.ListCategoriesResponse
	listCategoriesErr      error
}

func (c *fakeTransactionClient) CreateCategory(
	_ context.Context,
	in *transactionv1.CreateCategoryRequest,
	_ ...grpc.CallOption,
) (*transactionv1.CreateCategoryResponse, error) {
	c.createCategoryRequest = in
	if c.createCategoryErr != nil {
		return nil, c.createCategoryErr
	}

	return c.createCategoryResponse, nil
}

func (c *fakeTransactionClient) ListCategories(
	_ context.Context,
	in *transactionv1.ListCategoriesRequest,
	_ ...grpc.CallOption,
) (*transactionv1.ListCategoriesResponse, error) {
	c.listCategoriesRequest = in
	if c.listCategoriesErr != nil {
		return nil, c.listCategoriesErr
	}

	return c.listCategoriesResponse, nil
}

func (c *fakeTransactionClient) CreateTransaction(
	context.Context,
	*transactionv1.CreateTransactionRequest,
	...grpc.CallOption,
) (*transactionv1.CreateTransactionResponse, error) {
	return &transactionv1.CreateTransactionResponse{}, nil
}

func (c *fakeTransactionClient) ListTransactions(
	context.Context,
	*transactionv1.ListTransactionsRequest,
	...grpc.CallOption,
) (*transactionv1.ListTransactionsResponse, error) {
	return &transactionv1.ListTransactionsResponse{}, nil
}

func (c *fakeTransactionClient) GetBalance(
	context.Context,
	*transactionv1.GetBalanceRequest,
	...grpc.CallOption,
) (*transactionv1.GetBalanceResponse, error) {
	return &transactionv1.GetBalanceResponse{}, nil
}
