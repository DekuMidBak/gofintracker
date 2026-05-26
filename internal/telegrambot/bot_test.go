package telegrambot

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestBotResponseForText(t *testing.T) {
	bot := NewBot(&fakeTelegramAPI{}, &fakeGatewayAPI{}, slog.Default())

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "start",
			text: "/start",
			want: StartMessage,
		},
		{
			name: "help",
			text: "/help",
			want: HelpMessage,
		},
		{
			name: "unknown",
			text: "/unknown",
			want: "Неизвестная команда. Напиши /help, чтобы посмотреть список команд.",
		},
		{
			name: "future command",
			text: "/balance",
			want: loginRequiredMessage(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bot.ResponseForText(context.Background(), 100, tt.text)
			if got != tt.want {
				t.Fatalf("unexpected response:\nwant: %q\n got: %q", tt.want, got)
			}
		})
	}
}

func TestBotHandleUpdateSendsResponse(t *testing.T) {
	telegram := &fakeTelegramAPI{}
	bot := NewBot(telegram, &fakeGatewayAPI{}, slog.Default())

	err := bot.HandleUpdate(context.Background(), Update{
		ID: 1,
		Message: &Message{
			Text: "/start",
			Chat: Chat{ID: 100},
		},
	})
	if err != nil {
		t.Fatalf("handle update: %v", err)
	}

	if telegram.sentChatID != 100 || telegram.sentText != StartMessage {
		t.Fatalf("unexpected sent message: chat_id=%d text=%q", telegram.sentChatID, telegram.sentText)
	}
}

func TestBotHandleUpdateIgnoresNonTextUpdates(t *testing.T) {
	telegram := &fakeTelegramAPI{}
	bot := NewBot(telegram, &fakeGatewayAPI{}, slog.Default())

	if err := bot.HandleUpdate(context.Background(), Update{ID: 1}); err != nil {
		t.Fatalf("handle update: %v", err)
	}

	if telegram.sentText != "" {
		t.Fatalf("expected no message, got %q", telegram.sentText)
	}
}

func TestBotLoginStoresSession(t *testing.T) {
	gateway := &fakeGatewayAPI{
		loginResult: AuthResult{
			UserID:      "user-1",
			AccessToken: "token-1",
		},
	}
	bot := NewBot(&fakeTelegramAPI{}, gateway, slog.Default())

	response := bot.ResponseForText(context.Background(), 100, "/login user@example.com secret")
	if response != "Готово, вход выполнен." {
		t.Fatalf("unexpected response: %q", response)
	}

	if bot.sessions[100] != "token-1" {
		t.Fatalf("expected token to be stored")
	}

	if gateway.loginEmail != "user@example.com" || gateway.loginPassword != "secret" {
		t.Fatalf("unexpected login args: email=%q password=%q", gateway.loginEmail, gateway.loginPassword)
	}
}

func TestBotBalanceUsesSessionToken(t *testing.T) {
	gateway := &fakeGatewayAPI{
		balances: []Balance{
			{
				Currency:      "RUB",
				IncomeAmount:  100000,
				ExpenseAmount: 25000,
				BalanceAmount: 75000,
			},
		},
	}
	bot := NewBot(&fakeTelegramAPI{}, gateway, slog.Default())
	bot.sessions[100] = "token-1"

	response := bot.ResponseForText(context.Background(), 100, "/balance")
	if !strings.Contains(response, "RUB: доходы 100000, расходы 25000, итог 75000") {
		t.Fatalf("unexpected response: %q", response)
	}

	if gateway.balanceToken != "token-1" {
		t.Fatalf("expected balance token to be forwarded")
	}
}

func TestBotAddCategory(t *testing.T) {
	gateway := &fakeGatewayAPI{
		createdCategory: Category{
			ID:   "category-1",
			Name: "Food",
			Type: "expense",
		},
	}
	bot := NewBot(&fakeTelegramAPI{}, gateway, slog.Default())
	bot.sessions[100] = "token-1"

	response := bot.ResponseForText(context.Background(), 100, "/add_category expense Food")
	if response != "Категория создана: Food (expense)." {
		t.Fatalf("unexpected response: %q", response)
	}

	if gateway.createCategoryName != "Food" || gateway.createCategoryType != "expense" {
		t.Fatalf("unexpected category args: name=%q type=%q", gateway.createCategoryName, gateway.createCategoryType)
	}
}

func TestBotAddExpenseFindsCategoryAndCreatesTransaction(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	gateway := &fakeGatewayAPI{
		categories: []Category{
			{
				ID:   "category-1",
				Name: "Food",
				Type: "expense",
			},
		},
		createdTransaction: Transaction{
			ID:         "transaction-1",
			CategoryID: "category-1",
			Type:       "expense",
			Amount:     1500,
			Currency:   "RUB",
		},
	}
	bot := NewBot(&fakeTelegramAPI{}, gateway, slog.Default())
	bot.sessions[100] = "token-1"
	bot.now = func() time.Time {
		return now
	}

	response := bot.ResponseForText(context.Background(), 100, "/add_expense 1500 Food lunch")
	if response != "Операция добавлена: expense 1500 RUB в категории Food." {
		t.Fatalf("unexpected response: %q", response)
	}

	if gateway.createdTransactionParams.CategoryID != "category-1" ||
		gateway.createdTransactionParams.Amount != 1500 ||
		gateway.createdTransactionParams.OccurredAt != now {
		t.Fatalf("unexpected transaction params: %+v", gateway.createdTransactionParams)
	}
}

func TestBotMonthlyUsesCommandPeriod(t *testing.T) {
	gateway := &fakeGatewayAPI{
		monthlySummaries: []MonthlySummary{
			{
				Currency:      "RUB",
				IncomeAmount:  100000,
				ExpenseAmount: 25000,
				BalanceAmount: 75000,
			},
		},
	}
	bot := NewBot(&fakeTelegramAPI{}, gateway, slog.Default())
	bot.sessions[100] = "token-1"

	response := bot.ResponseForText(context.Background(), 100, "/monthly 2026 5")
	if !strings.Contains(response, "Статистика за 2026-05") {
		t.Fatalf("unexpected response: %q", response)
	}

	if gateway.monthlyYear != 2026 || gateway.monthlyMonth != 5 {
		t.Fatalf("unexpected period: year=%d month=%d", gateway.monthlyYear, gateway.monthlyMonth)
	}
}

func TestBotCategoryStatsUsesCommandFilters(t *testing.T) {
	gateway := &fakeGatewayAPI{
		categories: []Category{
			{
				ID:   "category-1",
				Name: "Food",
				Type: "expense",
			},
		},
		categoryStats: []CategoryStat{
			{
				CategoryID: "category-1",
				Currency:   "RUB",
				Type:       "expense",
				Amount:     25000,
			},
		},
	}
	bot := NewBot(&fakeTelegramAPI{}, gateway, slog.Default())
	bot.sessions[100] = "token-1"

	response := bot.ResponseForText(context.Background(), 100, "/category_stats 2026 5 expense")
	if !strings.Contains(response, "Food: expense 25000 RUB") {
		t.Fatalf("unexpected response: %q", response)
	}

	if gateway.categoryStatsYear != 2026 ||
		gateway.categoryStatsMonth != 5 ||
		gateway.categoryStatsType != "expense" {
		t.Fatalf(
			"unexpected filters: year=%d month=%d type=%q",
			gateway.categoryStatsYear,
			gateway.categoryStatsMonth,
			gateway.categoryStatsType,
		)
	}
}

type fakeTelegramAPI struct {
	updates []Update

	sentChatID int64
	sentText   string
}

func (api *fakeTelegramAPI) GetUpdates(context.Context, int64, time.Duration) ([]Update, error) {
	return api.updates, nil
}

func (api *fakeTelegramAPI) SendMessage(_ context.Context, chatID int64, text string) error {
	api.sentChatID = chatID
	api.sentText = text
	return nil
}

type fakeGatewayAPI struct {
	registerResult AuthResult
	registerEmail  string
	registerPass   string

	loginResult   AuthResult
	loginEmail    string
	loginPassword string

	categories []Category

	createdCategory     Category
	createCategoryName  string
	createCategoryType  string
	createCategoryToken string

	createdTransaction       Transaction
	createdTransactionParams CreateTransactionParams

	balances     []Balance
	balanceToken string

	monthlySummaries []MonthlySummary
	monthlyYear      int
	monthlyMonth     int

	categoryStats      []CategoryStat
	categoryStatsYear  int
	categoryStatsMonth int
	categoryStatsType  string
}

func (api *fakeGatewayAPI) Register(_ context.Context, email string, password string) (AuthResult, error) {
	api.registerEmail = email
	api.registerPass = password
	return api.registerResult, nil
}

func (api *fakeGatewayAPI) Login(_ context.Context, email string, password string) (AuthResult, error) {
	api.loginEmail = email
	api.loginPassword = password
	return api.loginResult, nil
}

func (api *fakeGatewayAPI) ListCategories(context.Context, string) ([]Category, error) {
	return api.categories, nil
}

func (api *fakeGatewayAPI) CreateCategory(
	_ context.Context,
	accessToken string,
	name string,
	transactionType string,
) (Category, error) {
	api.createCategoryToken = accessToken
	api.createCategoryName = name
	api.createCategoryType = transactionType
	return api.createdCategory, nil
}

func (api *fakeGatewayAPI) CreateTransaction(
	_ context.Context,
	_ string,
	params CreateTransactionParams,
) (Transaction, error) {
	api.createdTransactionParams = params
	return api.createdTransaction, nil
}

func (api *fakeGatewayAPI) GetBalance(_ context.Context, accessToken string) ([]Balance, error) {
	api.balanceToken = accessToken
	return api.balances, nil
}

func (api *fakeGatewayAPI) GetMonthlySummary(_ context.Context, _ string, year int, month int) ([]MonthlySummary, error) {
	api.monthlyYear = year
	api.monthlyMonth = month
	return api.monthlySummaries, nil
}

func (api *fakeGatewayAPI) GetCategoryStats(
	_ context.Context,
	_ string,
	year int,
	month int,
	transactionType string,
) ([]CategoryStat, error) {
	api.categoryStatsYear = year
	api.categoryStatsMonth = month
	api.categoryStatsType = transactionType
	return api.categoryStats, nil
}
