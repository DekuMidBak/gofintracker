package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/DekuMidBak/gofintracker/internal/analytics"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	checkViolation              = "23514"
	invalidTextRepresentation   = "22P02"
	invalidBinaryRepresentation = "22P03"
)

type Repository struct {
	pool *pgxpool.Pool
}

var _ analytics.Repository = (*Repository)(nil)

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ApplyTransactionCreated(
	ctx context.Context,
	event analytics.TransactionCreated,
) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin analytics transaction: %w", err)
	}
	defer rollback(ctx, tx)

	applied, err := insertProcessedEvent(ctx, tx, event.EventID)
	if err != nil {
		return false, err
	}

	if !applied {
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit duplicate analytics event: %w", err)
		}

		return false, nil
	}

	year, month := eventPeriod(event)
	if err := upsertMonthlyAggregate(ctx, tx, event, year, month); err != nil {
		return false, err
	}

	if err := upsertCategoryAggregate(ctx, tx, event, year, month); err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit analytics transaction: %w", err)
	}

	return true, nil
}

func (r *Repository) GetMonthlySummary(
	ctx context.Context,
	userID string,
	year int,
	month int,
) ([]analytics.MonthlySummary, error) {
	const query = `
		SELECT currency, income_amount, expense_amount
		FROM monthly_aggregates
		WHERE user_id = $1::uuid
			AND year = $2
			AND month = $3
		ORDER BY currency ASC
	`

	rows, err := r.pool.Query(ctx, query, userID, year, month)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	summaries := make([]analytics.MonthlySummary, 0)
	for rows.Next() {
		var summary analytics.MonthlySummary
		if err := rows.Scan(
			&summary.Currency,
			&summary.IncomeAmount,
			&summary.ExpenseAmount,
		); err != nil {
			return nil, mapError(err)
		}

		summary.BalanceAmount = summary.IncomeAmount - summary.ExpenseAmount
		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}

	return summaries, nil
}

func (r *Repository) GetCategoryStats(
	ctx context.Context,
	filter analytics.CategoryStatsFilter,
) ([]analytics.CategoryStat, error) {
	query := `
		SELECT category_id::text, currency, type, amount
		FROM category_aggregates
		WHERE user_id = $1::uuid
			AND year = $2
			AND month = $3
	`
	args := []any{filter.UserID, filter.Year, filter.Month}

	if filter.Type != nil {
		args = append(args, string(*filter.Type))
		query += fmt.Sprintf(" AND type = $%d", len(args))
	}

	query += " ORDER BY amount DESC, category_id ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	stats := make([]analytics.CategoryStat, 0)
	for rows.Next() {
		var stat analytics.CategoryStat
		var statType string
		if err := rows.Scan(
			&stat.CategoryID,
			&stat.Currency,
			&statType,
			&stat.Amount,
		); err != nil {
			return nil, mapError(err)
		}

		stat.Type = analytics.TransactionType(statType)
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}

	return stats, nil
}

func insertProcessedEvent(ctx context.Context, tx pgx.Tx, eventID string) (bool, error) {
	const query = `
		INSERT INTO processed_events (event_id)
		VALUES ($1::uuid)
		ON CONFLICT (event_id) DO NOTHING
	`

	tag, err := tx.Exec(ctx, query, eventID)
	if err != nil {
		return false, mapError(err)
	}

	return tag.RowsAffected() == 1, nil
}

func upsertMonthlyAggregate(
	ctx context.Context,
	tx pgx.Tx,
	event analytics.TransactionCreated,
	year int,
	month int,
) error {
	const query = `
		INSERT INTO monthly_aggregates (
			user_id,
			year,
			month,
			currency,
			income_amount,
			expense_amount
		)
		VALUES (
			$1::uuid,
			$2,
			$3,
			$4,
			CASE WHEN $5 = 'income' THEN $6 ELSE 0 END,
			CASE WHEN $5 = 'expense' THEN $6 ELSE 0 END
		)
		ON CONFLICT (user_id, year, month, currency)
		DO UPDATE SET
			income_amount = monthly_aggregates.income_amount + EXCLUDED.income_amount,
			expense_amount = monthly_aggregates.expense_amount + EXCLUDED.expense_amount,
			updated_at = now()
	`

	if _, err := tx.Exec(
		ctx,
		query,
		event.UserID,
		year,
		month,
		event.Currency,
		string(event.Type),
		event.Amount,
	); err != nil {
		return mapError(err)
	}

	return nil
}

func upsertCategoryAggregate(
	ctx context.Context,
	tx pgx.Tx,
	event analytics.TransactionCreated,
	year int,
	month int,
) error {
	const query = `
		INSERT INTO category_aggregates (
			user_id,
			category_id,
			year,
			month,
			currency,
			type,
			amount
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, category_id, year, month, currency, type)
		DO UPDATE SET
			amount = category_aggregates.amount + EXCLUDED.amount,
			updated_at = now()
	`

	if _, err := tx.Exec(
		ctx,
		query,
		event.UserID,
		event.CategoryID,
		year,
		month,
		event.Currency,
		string(event.Type),
		event.Amount,
	); err != nil {
		return mapError(err)
	}

	return nil
}

func eventPeriod(event analytics.TransactionCreated) (int, int) {
	occurredAt := event.OccurredAt.UTC()

	return occurredAt.Year(), int(occurredAt.Month())
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func mapError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case checkViolation:
			return mapCheckViolation(pgErr.ConstraintName)
		case invalidTextRepresentation, invalidBinaryRepresentation:
			return analytics.ErrInvalidID
		}
	}

	return fmt.Errorf("analytics postgres repository: %w", err)
}

func mapCheckViolation(constraintName string) error {
	switch constraintName {
	case "monthly_aggregates_month_check",
		"monthly_aggregates_year_check",
		"category_aggregates_month_check",
		"category_aggregates_year_check":
		return analytics.ErrInvalidPeriod
	case "monthly_aggregates_currency_check",
		"category_aggregates_currency_check":
		return analytics.ErrInvalidCurrency
	case "monthly_aggregates_income_non_negative_check",
		"monthly_aggregates_expense_non_negative_check",
		"category_aggregates_amount_non_negative_check":
		return analytics.ErrInvalidAmount
	case "category_aggregates_type_check":
		return analytics.ErrInvalidType
	default:
		return fmt.Errorf("analytics check violation: %s", constraintName)
	}
}
