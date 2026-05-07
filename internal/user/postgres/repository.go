package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/DekuMidBak/gofintracker/internal/user"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	uniqueViolation             = "23505"
	invalidTextRepresentation   = "22P02"
	invalidBinaryRepresentation = "22P03"
)

type Repository struct {
	pool *pgxpool.Pool
}

var _ user.Repository = (*Repository)(nil)

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, params user.CreateUserParams) (user.User, error) {
	const query = `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id::text, email, password_hash, created_at, updated_at
	`

	created, err := scanUser(r.pool.QueryRow(ctx, query, params.Email, params.PasswordHash))
	if err != nil {
		return user.User{}, mapError(err)
	}

	return created, nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (user.User, error) {
	const query = `
		SELECT id::text, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1::uuid
	`

	found, err := scanUser(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		return user.User{}, mapError(err)
	}

	return found, nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (user.User, error) {
	const query = `
		SELECT id::text, email, password_hash, created_at, updated_at
		FROM users
		WHERE lower(email) = lower($1)
	`

	found, err := scanUser(r.pool.QueryRow(ctx, query, email))
	if err != nil {
		return user.User{}, mapError(err)
	}

	return found, nil
}

func scanUser(row pgx.Row) (user.User, error) {
	var found user.User
	if err := row.Scan(
		&found.ID,
		&found.Email,
		&found.PasswordHash,
		&found.CreatedAt,
		&found.UpdatedAt,
	); err != nil {
		return user.User{}, err
	}

	return found, nil
}

func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return user.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			return user.ErrEmailExists
		case invalidTextRepresentation, invalidBinaryRepresentation:
			return user.ErrInvalidUserID
		}
	}

	return fmt.Errorf("user postgres repository: %w", err)
}
