package telegrambot

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestBotResponseForText(t *testing.T) {
	bot := NewBot(&fakeTelegramAPI{}, slog.Default())

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
			want: "Эта команда уже распознается, но будет подключена к финансовому API следующим шагом.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bot.ResponseForText(tt.text)
			if got != tt.want {
				t.Fatalf("unexpected response:\nwant: %q\n got: %q", tt.want, got)
			}
		})
	}
}

func TestBotHandleUpdateSendsResponse(t *testing.T) {
	telegram := &fakeTelegramAPI{}
	bot := NewBot(telegram, slog.Default())

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
	bot := NewBot(telegram, slog.Default())

	if err := bot.HandleUpdate(context.Background(), Update{ID: 1}); err != nil {
		t.Fatalf("handle update: %v", err)
	}

	if telegram.sentText != "" {
		t.Fatalf("expected no message, got %q", telegram.sentText)
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
