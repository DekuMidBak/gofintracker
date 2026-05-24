package telegrambot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	CommandStart         = "start"
	CommandHelp          = "help"
	CommandRegister      = "register"
	CommandLogin         = "login"
	CommandCategories    = "categories"
	CommandAddCategory   = "add_category"
	CommandAddExpense    = "add_expense"
	CommandAddIncome     = "add_income"
	CommandBalance       = "balance"
	CommandMonthly       = "monthly"
	CommandCategoryStats = "category_stats"
)

const DefaultCurrency = "RUB"

var (
	ErrEmptyCommand   = errors.New("empty command")
	ErrUnknownCommand = errors.New("unknown command")
	ErrInvalidCommand = errors.New("invalid command")
)

type Command struct {
	Name            string
	Email           string
	Password        string
	TransactionType string
	CategoryName    string
	Amount          int64
	Description     string
	Year            int
	Month           int
	Currency        string
}

func ParseCommand(text string) (Command, error) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return Command{}, ErrEmptyCommand
	}

	name := commandName(fields[0])
	args := fields[1:]

	switch name {
	case CommandStart, CommandHelp, CommandCategories, CommandBalance:
		if len(args) != 0 {
			return Command{}, fmt.Errorf("%w: /%s does not accept arguments", ErrInvalidCommand, name)
		}
		return Command{Name: name, Currency: DefaultCurrency}, nil
	case CommandRegister, CommandLogin:
		return parseAuthCommand(name, args)
	case CommandAddCategory:
		return parseAddCategoryCommand(args)
	case CommandAddExpense:
		return parseTransactionCommand(CommandAddExpense, "expense", args)
	case CommandAddIncome:
		return parseTransactionCommand(CommandAddIncome, "income", args)
	case CommandMonthly:
		return parseMonthlyCommand(args)
	case CommandCategoryStats:
		return parseCategoryStatsCommand(args)
	default:
		return Command{}, fmt.Errorf("%w: %s", ErrUnknownCommand, name)
	}
}

func commandName(value string) string {
	value = strings.TrimPrefix(value, "/")
	if index := strings.Index(value, "@"); index >= 0 {
		value = value[:index]
	}

	return strings.ToLower(value)
}

func parseAuthCommand(name string, args []string) (Command, error) {
	if len(args) != 2 {
		return Command{}, fmt.Errorf("%w: /%s requires email and password", ErrInvalidCommand, name)
	}

	return Command{
		Name:     name,
		Email:    args[0],
		Password: args[1],
		Currency: DefaultCurrency,
	}, nil
}

func parseAddCategoryCommand(args []string) (Command, error) {
	if len(args) < 2 {
		return Command{}, fmt.Errorf("%w: /add_category requires type and name", ErrInvalidCommand)
	}

	transactionType := args[0]
	if !isTransactionType(transactionType) {
		return Command{}, fmt.Errorf("%w: transaction type must be income or expense", ErrInvalidCommand)
	}

	return Command{
		Name:            CommandAddCategory,
		TransactionType: transactionType,
		CategoryName:    strings.Join(args[1:], " "),
		Currency:        DefaultCurrency,
	}, nil
}

func parseTransactionCommand(name, transactionType string, args []string) (Command, error) {
	if len(args) < 2 {
		return Command{}, fmt.Errorf("%w: /%s requires amount and category name", ErrInvalidCommand, name)
	}

	amount, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || amount <= 0 {
		return Command{}, fmt.Errorf("%w: amount must be a positive integer", ErrInvalidCommand)
	}

	description := ""
	if len(args) > 2 {
		description = strings.Join(args[2:], " ")
	}

	return Command{
		Name:            name,
		TransactionType: transactionType,
		Amount:          amount,
		CategoryName:    args[1],
		Description:     description,
		Currency:        DefaultCurrency,
	}, nil
}

func parseMonthlyCommand(args []string) (Command, error) {
	if len(args) != 2 {
		return Command{}, fmt.Errorf("%w: /monthly requires year and month", ErrInvalidCommand)
	}

	year, month, err := parseYearMonth(args[0], args[1])
	if err != nil {
		return Command{}, err
	}

	return Command{
		Name:     CommandMonthly,
		Year:     year,
		Month:    month,
		Currency: DefaultCurrency,
	}, nil
}

func parseCategoryStatsCommand(args []string) (Command, error) {
	if len(args) != 3 {
		return Command{}, fmt.Errorf("%w: /category_stats requires year, month and type", ErrInvalidCommand)
	}

	year, month, err := parseYearMonth(args[0], args[1])
	if err != nil {
		return Command{}, err
	}

	transactionType := args[2]
	if !isTransactionType(transactionType) {
		return Command{}, fmt.Errorf("%w: transaction type must be income or expense", ErrInvalidCommand)
	}

	return Command{
		Name:            CommandCategoryStats,
		Year:            year,
		Month:           month,
		TransactionType: transactionType,
		Currency:        DefaultCurrency,
	}, nil
}

func parseYearMonth(rawYear, rawMonth string) (int, int, error) {
	year, err := strconv.Atoi(rawYear)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: year must be an integer", ErrInvalidCommand)
	}

	month, err := strconv.Atoi(rawMonth)
	if err != nil || month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("%w: month must be between 1 and 12", ErrInvalidCommand)
	}

	return year, month, nil
}

func isTransactionType(value string) bool {
	return value == "income" || value == "expense"
}
