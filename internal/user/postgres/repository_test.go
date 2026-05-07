package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DekuMidBak/gofintracker/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryCreateAndFindByID(t *testing.T) {
	repo, pool := newTestRepository(t)
	ctx := testContext(t)

	email := uniqueEmail("create-find-id")
	created, err := repo.Create(ctx, user.CreateUserParams{
		Email:        email,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		deleteUser(t, pool, created.ID)
	})

	if created.ID == "" {
		t.Fatal("expected generated user id")
	}

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("find user by id: %v", err)
	}

	if found.Email != email {
		t.Fatalf("expected email %q, got %q", email, found.Email)
	}

	if found.PasswordHash != "hash" {
		t.Fatalf("expected password hash to be returned")
	}
}

func TestRepositoryFindByEmailIsCaseInsensitive(t *testing.T) {
	repo, pool := newTestRepository(t)
	ctx := testContext(t)

	email := uniqueEmail("case-insensitive")
	created, err := repo.Create(ctx, user.CreateUserParams{
		Email:        email,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		deleteUser(t, pool, created.ID)
	})

	found, err := repo.FindByEmail(ctx, upperEmail(email))
	if err != nil {
		t.Fatalf("find user by email: %v", err)
	}

	if found.ID != created.ID {
		t.Fatalf("expected user id %q, got %q", created.ID, found.ID)
	}
}

func TestRepositoryCreateDuplicateEmailReturnsErrEmailExists(t *testing.T) {
	repo, pool := newTestRepository(t)
	ctx := testContext(t)

	email := uniqueEmail("duplicate")
	created, err := repo.Create(ctx, user.CreateUserParams{
		Email:        email,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		deleteUser(t, pool, created.ID)
	})

	_, err = repo.Create(ctx, user.CreateUserParams{
		Email:        upperEmail(email),
		PasswordHash: "another-hash",
	})
	if !errors.Is(err, user.ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
}

func TestRepositoryFindByEmailReturnsErrNotFound(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := testContext(t)

	_, err := repo.FindByEmail(ctx, uniqueEmail("missing"))
	if !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRepositoryFindByIDReturnsErrInvalidUserID(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := testContext(t)

	_, err := repo.FindByID(ctx, "not-a-uuid")
	if !errors.Is(err, user.ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func newTestRepository(t *testing.T) (*Repository, *pgxpool.Pool) {
	t.Helper()

	dsn := os.Getenv("USERS_TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("USERS_TEST_DATABASE_DSN is not set")
	}

	ctx := testContext(t)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	return New(pool), pool
}

func testContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	return ctx
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@example.test", prefix, time.Now().UnixNano())
}

func upperEmail(email string) string {
	return strings.ToUpper(email)
}

func deleteUser(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx, "DELETE FROM users WHERE id = $1::uuid", id); err != nil {
		t.Fatalf("delete test user: %v", err)
	}
}
