package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/DekuMidBak/gofintracker/internal/transaction"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	uniqueViolation             = "23505"
	foreignKeyViolation         = "23503"
	invalidTextRepresentation   = "22P02"
	invalidBinaryRepresentation = "22P03"
)

type Repository struct {
	pool *pgxpool.Pool
}

var _ transaction.Repository = (*Repository)(nil)

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateCategory(ctx context.Context, params transaction.CreateCategoryParams) (transaction.Category, error) {
	const query = `
		INSERT INTO categories (user_id, name, type)
		VALUES ($1::uuid, $2, $3)
		RETURNING id::text, user_id::text, name, type, created_at
	`

	category, err := scanCategory(r.pool.QueryRow(ctx, query, params.UserID, params.Name, string(params.Type)))
	if err != nil {
		return transaction.Category{}, mapError(err)
	}

	return category, nil
}

func (r *Repository) ListCategories(ctx context.Context, filter transaction.ListCategoriesFilter) ([]transaction.Category, error) {
	query := `
		SELECT id::text, user_id::text, name, type, created_at
		FROM categories
		WHERE user_id = $1::uuid
	`
	args := []any{filter.UserID}

	if filter.Type != nil {
		args = append(args, string(*filter.Type))
		query += fmt.Sprintf(" AND type = $%d", len(args))
	}

	query += " ORDER BY name ASC, created_at ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	categories := make([]transaction.Category, 0)
	for rows.Next() {
		category, err := scanCategory(rows)
		if err != nil {
			return nil, mapError(err)
		}

		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}

	return categories, nil
}

func (r *Repository) CreateTransaction(ctx context.Context, params transaction.CreateTransactionParams) (transaction.Transaction, error) {
	const query = `
		INSERT INTO transactions (user_id, category_id, type, amount, currency, description, occurred_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
		RETURNING id::text, user_id::text, category_id::text, type, amount, currency, description, occurred_at, created_at
	`

	created, err := scanTransaction(
		r.pool.QueryRow(
			ctx,
			query,
			params.UserID,
			params.CategoryID,
			string(params.Type),
			params.Amount,
			params.Currency,
			params.Description,
			params.OccurredAt,
		),
	)
	if err != nil {
		return transaction.Transaction{}, mapError(err)
	}

	return created, nil
}

func (r *Repository) ListTransactions(ctx context.Context, filter transaction.ListTransactionsFilter) (transaction.ListTransactionsResult, error) {
	query := `
		SELECT
			id::text,
			user_id::text,
			category_id::text,
			type,
			amount,
			currency,
			description,
			occurred_at,
			created_at,
			COUNT(*) OVER()
		FROM transactions
		WHERE user_id = $1::uuid
	`
	args := []any{filter.UserID}

	if filter.From != nil {
		args = append(args, *filter.From)
		query += fmt.Sprintf(" AND occurred_at >= $%d", len(args))
	}

	if filter.To != nil {
		args = append(args, *filter.To)
		query += fmt.Sprintf(" AND occurred_at <= $%d", len(args))
	}

	if filter.CategoryID != nil {
		args = append(args, *filter.CategoryID)
		query += fmt.Sprintf(" AND category_id = $%d::uuid", len(args))
	}

	if filter.Type != nil {
		args = append(args, string(*filter.Type))
		query += fmt.Sprintf(" AND type = $%d", len(args))
	}

	query += " ORDER BY occurred_at DESC, created_at DESC"

	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return transaction.ListTransactionsResult{}, mapError(err)
	}
	defer rows.Close()

	result := transaction.ListTransactionsResult{
		Transactions: make([]transaction.Transaction, 0),
	}

	for rows.Next() {
		var item transaction.Transaction
		var itemType string

		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.CategoryID,
			&itemType,
			&item.Amount,
			&item.Currency,
			&item.Description,
			&item.OccurredAt,
			&item.CreatedAt,
			&result.TotalCount,
		); err != nil {
			return transaction.ListTransactionsResult{}, mapError(err)
		}

		item.Type = transaction.Type(itemType)
		result.Transactions = append(result.Transactions, item)
	}

	if err := rows.Err(); err != nil {
		return transaction.ListTransactionsResult{}, mapError(err)
	}

	return result, nil
}

func (r *Repository) GetBalance(ctx context.Context, userID string) ([]transaction.CurrencyBalance, error) {
	const query = `
		SELECT
			currency,
			COALESCE(SUM(amount) FILTER (WHERE type = 'income'), 0) AS income_amount,
			COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0) AS expense_amount
		FROM transactions
		WHERE user_id = $1::uuid
		GROUP BY currency
		ORDER BY currency ASC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	balances := make([]transaction.CurrencyBalance, 0)
	for rows.Next() {
		var balance transaction.CurrencyBalance
		if err := rows.Scan(
			&balance.Currency,
			&balance.IncomeAmount,
			&balance.ExpenseAmount,
		); err != nil {
			return nil, mapError(err)
		}

		balance.BalanceAmount = balance.IncomeAmount - balance.ExpenseAmount
		balances = append(balances, balance)
	}

	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}

	return balances, nil
}

func scanCategory(row pgx.Row) (transaction.Category, error) {
	var category transaction.Category
	var categoryType string

	if err := row.Scan(
		&category.ID,
		&category.UserID,
		&category.Name,
		&categoryType,
		&category.CreatedAt,
	); err != nil {
		return transaction.Category{}, err
	}

	category.Type = transaction.Type(categoryType)
	return category, nil
}

func scanTransaction(row pgx.Row) (transaction.Transaction, error) {
	var item transaction.Transaction
	var itemType string

	if err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.CategoryID,
		&itemType,
		&item.Amount,
		&item.Currency,
		&item.Description,
		&item.OccurredAt,
		&item.CreatedAt,
	); err != nil {
		return transaction.Transaction{}, err
	}

	item.Type = transaction.Type(itemType)
	return item, nil
}

func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return transaction.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			if strings.Contains(pgErr.ConstraintName, "categories") {
				return transaction.ErrDuplicateCategory
			}
		case foreignKeyViolation:
			return transaction.ErrNotFound
		case invalidTextRepresentation, invalidBinaryRepresentation:
			return transaction.ErrInvalidID
		}
	}

	return fmt.Errorf("transaction postgres repository: %w", err)
}
