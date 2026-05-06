package user

import (
	"context"
	"errors"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrEmailExists   = errors.New("user email already exists")
	ErrInvalidUserID = errors.New("invalid user id")
)

type CreateUserParams struct {
	Email        string
	PasswordHash string
}

type Repository interface {
	Create(ctx context.Context, params CreateUserParams) (User, error)
	FindByID(ctx context.Context, id string) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
}
