package telegrambot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GatewayClient struct {
	baseURL    string
	httpClient *http.Client
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e APIError) Error() string {
	return fmt.Sprintf("gateway returned status %d: %s", e.StatusCode, e.Message)
}

type AuthResult struct {
	UserID      string `json:"user_id"`
	AccessToken string `json:"access_token"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Transaction struct {
	ID          string    `json:"id"`
	CategoryID  string    `json:"category_id"`
	Type        string    `json:"type"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	Description string    `json:"description"`
	OccurredAt  time.Time `json:"occurred_at"`
}

type CreateTransactionParams struct {
	CategoryID  string
	Type        string
	Amount      int64
	Currency    string
	Description string
	OccurredAt  time.Time
}

type Balance struct {
	Currency      string `json:"currency"`
	IncomeAmount  int64  `json:"income_amount"`
	ExpenseAmount int64  `json:"expense_amount"`
	BalanceAmount int64  `json:"balance_amount"`
}

type MonthlySummary struct {
	Currency      string `json:"currency"`
	IncomeAmount  int64  `json:"income_amount"`
	ExpenseAmount int64  `json:"expense_amount"`
	BalanceAmount int64  `json:"balance_amount"`
}

type CategoryStat struct {
	CategoryID string `json:"category_id"`
	Currency   string `json:"currency"`
	Type       string `json:"type"`
	Amount     int64  `json:"amount"`
}

func NewGatewayClient(baseURL string, httpClient *http.Client) *GatewayClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &GatewayClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *GatewayClient) Register(ctx context.Context, email, password string) (AuthResult, error) {
	var result AuthResult
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"email":    email,
		"password": password,
	}, &result)

	return result, err
}

func (c *GatewayClient) Login(ctx context.Context, email, password string) (AuthResult, error) {
	var result AuthResult
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email":    email,
		"password": password,
	}, &result)

	return result, err
}

func (c *GatewayClient) ListCategories(ctx context.Context, accessToken string) ([]Category, error) {
	var result struct {
		Categories []Category `json:"categories"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/categories", accessToken, nil, &result)

	return result.Categories, err
}

func (c *GatewayClient) CreateCategory(ctx context.Context, accessToken, name, transactionType string) (Category, error) {
	var result Category
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/categories", accessToken, map[string]string{
		"name": name,
		"type": transactionType,
	}, &result)

	return result, err
}

func (c *GatewayClient) CreateTransaction(
	ctx context.Context,
	accessToken string,
	params CreateTransactionParams,
) (Transaction, error) {
	var result Transaction
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/transactions", accessToken, map[string]any{
		"category_id": params.CategoryID,
		"type":        params.Type,
		"amount":      params.Amount,
		"currency":    params.Currency,
		"description": params.Description,
		"occurred_at": params.OccurredAt.Format(time.RFC3339),
	}, &result)

	return result, err
}

func (c *GatewayClient) GetBalance(ctx context.Context, accessToken string) ([]Balance, error) {
	var result struct {
		Balances []Balance `json:"balances"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/balance", accessToken, nil, &result)

	return result.Balances, err
}

func (c *GatewayClient) GetMonthlySummary(
	ctx context.Context,
	accessToken string,
	year int,
	month int,
) ([]MonthlySummary, error) {
	path := fmt.Sprintf("/api/v1/analytics/monthly?year=%d&month=%d", year, month)

	var result struct {
		Summaries []MonthlySummary `json:"summaries"`
	}
	err := c.doJSON(ctx, http.MethodGet, path, accessToken, nil, &result)

	return result.Summaries, err
}

func (c *GatewayClient) GetCategoryStats(
	ctx context.Context,
	accessToken string,
	year int,
	month int,
	transactionType string,
) ([]CategoryStat, error) {
	query := url.Values{}
	query.Set("year", fmt.Sprintf("%d", year))
	query.Set("month", fmt.Sprintf("%d", month))
	if transactionType != "" {
		query.Set("type", transactionType)
	}

	var result struct {
		Stats []CategoryStat `json:"stats"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/analytics/categories?"+query.Encode(), accessToken, nil, &result)

	return result.Stats, err
}

func (c *GatewayClient) doJSON(
	ctx context.Context,
	method string,
	path string,
	accessToken string,
	body any,
	result any,
) error {
	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		requestBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, requestBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var errorBody struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorBody); err != nil {
			errorBody.Error = resp.Status
		}

		return APIError{
			StatusCode: resp.StatusCode,
			Message:    errorBody.Error,
		}
	}

	if result == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}
