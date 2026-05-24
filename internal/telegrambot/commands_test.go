package telegrambot

import (
	"errors"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name string
		text string
		want Command
	}{
		{
			name: "start",
			text: "/start",
			want: Command{Name: CommandStart, Currency: DefaultCurrency},
		},
		{
			name: "command with bot name",
			text: "/balance@GoFinTrackerBot",
			want: Command{Name: CommandBalance, Currency: DefaultCurrency},
		},
		{
			name: "register",
			text: "/register user@example.com secret",
			want: Command{
				Name:     CommandRegister,
				Email:    "user@example.com",
				Password: "secret",
				Currency: DefaultCurrency,
			},
		},
		{
			name: "add category",
			text: "/add_category expense Food and cafes",
			want: Command{
				Name:            CommandAddCategory,
				TransactionType: "expense",
				CategoryName:    "Food and cafes",
				Currency:        DefaultCurrency,
			},
		},
		{
			name: "add expense",
			text: "/add_expense 1500 Food lunch with friends",
			want: Command{
				Name:            CommandAddExpense,
				TransactionType: "expense",
				Amount:          1500,
				CategoryName:    "Food",
				Description:     "lunch with friends",
				Currency:        DefaultCurrency,
			},
		},
		{
			name: "add income",
			text: "/add_income 100000 Salary May salary",
			want: Command{
				Name:            CommandAddIncome,
				TransactionType: "income",
				Amount:          100000,
				CategoryName:    "Salary",
				Description:     "May salary",
				Currency:        DefaultCurrency,
			},
		},
		{
			name: "monthly",
			text: "/monthly 2026 5",
			want: Command{
				Name:     CommandMonthly,
				Year:     2026,
				Month:    5,
				Currency: DefaultCurrency,
			},
		},
		{
			name: "category stats",
			text: "/category_stats 2026 5 expense",
			want: Command{
				Name:            CommandCategoryStats,
				Year:            2026,
				Month:           5,
				TransactionType: "expense",
				Currency:        DefaultCurrency,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCommand(tt.text)
			if err != nil {
				t.Fatalf("parse command: %v", err)
			}

			if got != tt.want {
				t.Fatalf("unexpected command:\nwant: %+v\n got: %+v", tt.want, got)
			}
		})
	}
}

func TestParseCommandRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantErr error
	}{
		{
			name:    "empty",
			text:    " ",
			wantErr: ErrEmptyCommand,
		},
		{
			name:    "unknown",
			text:    "/wat",
			wantErr: ErrUnknownCommand,
		},
		{
			name:    "invalid auth args",
			text:    "/login user@example.com",
			wantErr: ErrInvalidCommand,
		},
		{
			name:    "invalid transaction type",
			text:    "/add_category other Food",
			wantErr: ErrInvalidCommand,
		},
		{
			name:    "invalid amount",
			text:    "/add_expense -1 Food",
			wantErr: ErrInvalidCommand,
		},
		{
			name:    "invalid month",
			text:    "/monthly 2026 13",
			wantErr: ErrInvalidCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCommand(tt.text)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
