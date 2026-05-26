package telegrambot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	StartMessage = "Привет! Я GoFinTracker bot. Я помогу вести доходы, расходы, баланс и аналитику."
	HelpMessage  = "Команды:\n/start\n/help\n/register email password\n/login email password\n/categories\n/add_category income|expense name\n/add_expense amount category description\n/add_income amount category description\n/balance\n/monthly year month\n/category_stats year month income|expense"
)

type TelegramAPI interface {
	GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]Update, error)
	SendMessage(ctx context.Context, chatID int64, text string) error
}

type GatewayAPI interface {
	Register(ctx context.Context, email string, password string) (AuthResult, error)
	Login(ctx context.Context, email string, password string) (AuthResult, error)
	ListCategories(ctx context.Context, accessToken string) ([]Category, error)
	CreateCategory(ctx context.Context, accessToken string, name string, transactionType string) (Category, error)
	CreateTransaction(ctx context.Context, accessToken string, params CreateTransactionParams) (Transaction, error)
	GetBalance(ctx context.Context, accessToken string) ([]Balance, error)
	GetMonthlySummary(ctx context.Context, accessToken string, year int, month int) ([]MonthlySummary, error)
	GetCategoryStats(ctx context.Context, accessToken string, year int, month int, transactionType string) ([]CategoryStat, error)
}

type Bot struct {
	telegram        TelegramAPI
	gateway         GatewayAPI
	logger          *slog.Logger
	sessions        map[int64]string
	pollTimeout     time.Duration
	retryDelay      time.Duration
	nextUpdateID    int64
	now             func() time.Time
	sleepAfterError func(context.Context, time.Duration) error
}

func NewBot(telegram TelegramAPI, gateway GatewayAPI, logger *slog.Logger) *Bot {
	if logger == nil {
		logger = slog.Default()
	}

	return &Bot{
		telegram:        telegram,
		gateway:         gateway,
		logger:          logger,
		sessions:        make(map[int64]string),
		pollTimeout:     30 * time.Second,
		retryDelay:      2 * time.Second,
		now:             time.Now,
		sleepAfterError: sleepContext,
	}
}

func (b *Bot) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		updates, err := b.telegram.GetUpdates(ctx, b.nextUpdateID, b.pollTimeout)
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}

			b.logger.Warn("failed to get telegram updates", "error", err)
			if err := b.sleepAfterError(ctx, b.retryDelay); err != nil {
				return nil
			}
			continue
		}

		for _, update := range updates {
			if update.ID >= b.nextUpdateID {
				b.nextUpdateID = update.ID + 1
			}

			if err := b.HandleUpdate(ctx, update); err != nil {
				b.logger.Warn("failed to handle telegram update", "update_id", update.ID, "error", err)
			}
		}
	}
}

func (b *Bot) HandleUpdate(ctx context.Context, update Update) error {
	if update.Message == nil || update.Message.Text == "" {
		return nil
	}

	chatID := update.Message.Chat.ID
	response := b.ResponseForText(ctx, chatID, update.Message.Text)
	if response == "" {
		return nil
	}

	return b.telegram.SendMessage(ctx, chatID, response)
}

func (b *Bot) ResponseForText(ctx context.Context, chatID int64, text string) string {
	command, err := ParseCommand(text)
	if err != nil {
		if errors.Is(err, ErrUnknownCommand) {
			return "Неизвестная команда. Напиши /help, чтобы посмотреть список команд."
		}
		if errors.Is(err, ErrInvalidCommand) {
			return "Некорректная команда. Напиши /help, чтобы посмотреть формат команд."
		}

		return ""
	}

	switch command.Name {
	case CommandStart:
		return StartMessage
	case CommandHelp:
		return HelpMessage
	case CommandRegister:
		return b.handleRegister(ctx, chatID, command)
	case CommandLogin:
		return b.handleLogin(ctx, chatID, command)
	case CommandCategories:
		return b.handleCategories(ctx, chatID)
	case CommandAddCategory:
		return b.handleAddCategory(ctx, chatID, command)
	case CommandAddExpense, CommandAddIncome:
		return b.handleAddTransaction(ctx, chatID, command)
	case CommandBalance:
		return b.handleBalance(ctx, chatID)
	case CommandMonthly:
		return b.handleMonthly(ctx, chatID, command)
	case CommandCategoryStats:
		return b.handleCategoryStats(ctx, chatID, command)
	default:
		return "Неизвестная команда. Напиши /help, чтобы посмотреть список команд."
	}
}

func (b *Bot) handleRegister(ctx context.Context, chatID int64, command Command) string {
	if b.gateway == nil {
		return "Финансовый API не настроен."
	}

	result, err := b.gateway.Register(ctx, command.Email, command.Password)
	if err != nil {
		return "Не удалось зарегистрироваться: " + userError(err)
	}

	b.sessions[chatID] = result.AccessToken
	return "Готово, регистрация успешна. Теперь можно добавлять категории и операции."
}

func (b *Bot) handleLogin(ctx context.Context, chatID int64, command Command) string {
	if b.gateway == nil {
		return "Финансовый API не настроен."
	}

	result, err := b.gateway.Login(ctx, command.Email, command.Password)
	if err != nil {
		return "Не удалось войти: " + userError(err)
	}

	b.sessions[chatID] = result.AccessToken
	return "Готово, вход выполнен."
}

func (b *Bot) handleCategories(ctx context.Context, chatID int64) string {
	token, ok := b.accessToken(chatID)
	if !ok {
		return loginRequiredMessage()
	}

	categories, err := b.gateway.ListCategories(ctx, token)
	if err != nil {
		return "Не удалось получить категории: " + userError(err)
	}
	if len(categories) == 0 {
		return "Категорий пока нет. Добавь первую через /add_category income|expense name."
	}

	var builder strings.Builder
	builder.WriteString("Категории:")
	for _, category := range categories {
		builder.WriteString(fmt.Sprintf("\n- %s (%s)", category.Name, category.Type))
	}

	return builder.String()
}

func (b *Bot) handleAddCategory(ctx context.Context, chatID int64, command Command) string {
	token, ok := b.accessToken(chatID)
	if !ok {
		return loginRequiredMessage()
	}

	category, err := b.gateway.CreateCategory(ctx, token, command.CategoryName, command.TransactionType)
	if err != nil {
		return "Не удалось создать категорию: " + userError(err)
	}

	return fmt.Sprintf("Категория создана: %s (%s).", category.Name, category.Type)
}

func (b *Bot) handleAddTransaction(ctx context.Context, chatID int64, command Command) string {
	token, ok := b.accessToken(chatID)
	if !ok {
		return loginRequiredMessage()
	}

	categories, err := b.gateway.ListCategories(ctx, token)
	if err != nil {
		return "Не удалось получить категории: " + userError(err)
	}

	category, ok := findCategory(categories, command.CategoryName, command.TransactionType)
	if !ok {
		return fmt.Sprintf("Категория %q для типа %s не найдена. Создай ее через /add_category %s %s.", command.CategoryName, command.TransactionType, command.TransactionType, command.CategoryName)
	}

	transaction, err := b.gateway.CreateTransaction(ctx, token, CreateTransactionParams{
		CategoryID:  category.ID,
		Type:        command.TransactionType,
		Amount:      command.Amount,
		Currency:    command.Currency,
		Description: command.Description,
		OccurredAt:  b.now().UTC(),
	})
	if err != nil {
		return "Не удалось добавить операцию: " + userError(err)
	}

	return fmt.Sprintf("Операция добавлена: %s %d %s в категории %s.", transaction.Type, transaction.Amount, transaction.Currency, category.Name)
}

func (b *Bot) handleBalance(ctx context.Context, chatID int64) string {
	token, ok := b.accessToken(chatID)
	if !ok {
		return loginRequiredMessage()
	}

	balances, err := b.gateway.GetBalance(ctx, token)
	if err != nil {
		return "Не удалось получить баланс: " + userError(err)
	}
	if len(balances) == 0 {
		return "Баланс пока пуст."
	}

	var builder strings.Builder
	builder.WriteString("Баланс:")
	for _, balance := range balances {
		builder.WriteString(fmt.Sprintf(
			"\n%s: доходы %d, расходы %d, итог %d",
			balance.Currency,
			balance.IncomeAmount,
			balance.ExpenseAmount,
			balance.BalanceAmount,
		))
	}

	return builder.String()
}

func (b *Bot) handleMonthly(ctx context.Context, chatID int64, command Command) string {
	token, ok := b.accessToken(chatID)
	if !ok {
		return loginRequiredMessage()
	}

	summaries, err := b.gateway.GetMonthlySummary(ctx, token, command.Year, command.Month)
	if err != nil {
		return "Не удалось получить месячную статистику: " + userError(err)
	}
	if len(summaries) == 0 {
		return "За этот месяц статистики пока нет."
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Статистика за %04d-%02d:", command.Year, command.Month))
	for _, summary := range summaries {
		builder.WriteString(fmt.Sprintf(
			"\n%s: доходы %d, расходы %d, итог %d",
			summary.Currency,
			summary.IncomeAmount,
			summary.ExpenseAmount,
			summary.BalanceAmount,
		))
	}

	return builder.String()
}

func (b *Bot) handleCategoryStats(ctx context.Context, chatID int64, command Command) string {
	token, ok := b.accessToken(chatID)
	if !ok {
		return loginRequiredMessage()
	}

	stats, err := b.gateway.GetCategoryStats(ctx, token, command.Year, command.Month, command.TransactionType)
	if err != nil {
		return "Не удалось получить статистику по категориям: " + userError(err)
	}
	if len(stats) == 0 {
		return "За этот месяц статистики по категориям пока нет."
	}

	categories, err := b.gateway.ListCategories(ctx, token)
	if err != nil {
		return "Не удалось получить категории: " + userError(err)
	}
	categoryNames := categoryNamesByID(categories)

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Категории за %04d-%02d:", command.Year, command.Month))
	for _, stat := range stats {
		categoryName := categoryNames[stat.CategoryID]
		if categoryName == "" {
			categoryName = stat.CategoryID
		}

		builder.WriteString(fmt.Sprintf(
			"\n%s: %s %d %s",
			categoryName,
			stat.Type,
			stat.Amount,
			stat.Currency,
		))
	}

	return builder.String()
}

func (b *Bot) accessToken(chatID int64) (string, bool) {
	if b.gateway == nil {
		return "", false
	}

	token, ok := b.sessions[chatID]
	return token, ok && token != ""
}

func findCategory(categories []Category, name string, transactionType string) (Category, bool) {
	for _, category := range categories {
		if strings.EqualFold(category.Name, name) && category.Type == transactionType {
			return category, true
		}
	}

	return Category{}, false
}

func categoryNamesByID(categories []Category) map[string]string {
	result := make(map[string]string, len(categories))
	for _, category := range categories {
		result[category.ID] = category.Name
	}

	return result
}

func loginRequiredMessage() string {
	return "Сначала выполни /login email password или /register email password."
}

func userError(err error) string {
	var apiErr APIError
	if errors.As(err, &apiErr) && apiErr.Message != "" {
		return apiErr.Message
	}

	return err.Error()
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
