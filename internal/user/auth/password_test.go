package auth

import (
	"errors"
	"testing"
)

func TestHashPasswordAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if hash == "secret" {
		t.Fatal("expected password hash to differ from plain password")
	}

	if err := CheckPassword(hash, "secret"); err != nil {
		t.Fatalf("check password: %v", err)
	}
}

func TestHashPasswordReturnsErrorForEmptyPassword(t *testing.T) {
	_, err := HashPassword("")
	if !errors.Is(err, ErrEmptyPassword) {
		t.Fatalf("expected ErrEmptyPassword, got %v", err)
	}
}

func TestCheckPasswordReturnsErrorForWrongPassword(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	err = CheckPassword(hash, "wrong")
	if !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected ErrInvalidPassword, got %v", err)
	}
}
