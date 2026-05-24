package telegrambot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

type Bot struct {
	telegram        TelegramAPI
	logger          *slog.Logger
	pollTimeout     time.Duration
	retryDelay      time.Duration
	nextUpdateID    int64
	sleepAfterError func(context.Context, time.Duration) error
}

func NewBot(telegram TelegramAPI, logger *slog.Logger) *Bot {
	if logger == nil {
		logger = slog.Default()
	}

	return &Bot{
		telegram:        telegram,
		logger:          logger,
		pollTimeout:     30 * time.Second,
		retryDelay:      2 * time.Second,
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
	response := b.ResponseForText(update.Message.Text)
	if response == "" {
		return nil
	}

	return b.telegram.SendMessage(ctx, chatID, response)
}

func (b *Bot) ResponseForText(text string) string {
	command, err := ParseCommand(text)
	if err != nil {
		if errors.Is(err, ErrUnknownCommand) {
			return "Неизвестная команда. Напиши /help, чтобы посмотреть список команд."
		}
		if errors.Is(err, ErrInvalidCommand) {
			return fmt.Sprintf("Некорректная команда: %v", err)
		}

		return ""
	}

	switch command.Name {
	case CommandStart:
		return StartMessage
	case CommandHelp:
		return HelpMessage
	default:
		return "Эта команда уже распознается, но будет подключена к финансовому API следующим шагом."
	}
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
