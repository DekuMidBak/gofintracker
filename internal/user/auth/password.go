package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmptyPassword   = errors.New("password is empty")
	ErrInvalidPassword = errors.New("invalid password")
)

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	return string(hash), nil
}

func CheckPassword(passwordHash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return ErrInvalidPassword
	}

	return nil
}
