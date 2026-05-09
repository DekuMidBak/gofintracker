package auth

import (
	"errors"
	"testing"
	"time"
)

func TestTokenManagerGenerateAndValidate(t *testing.T) {
	manager, err := NewTokenManager("secret", time.Minute)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}

	token, err := manager.Generate("user-id")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	userID, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}

	if userID != "user-id" {
		t.Fatalf("expected user-id, got %q", userID)
	}
}

func TestNewTokenManagerValidatesConfig(t *testing.T) {
	_, err := NewTokenManager("", time.Minute)
	if !errors.Is(err, ErrEmptyTokenSecret) {
		t.Fatalf("expected ErrEmptyTokenSecret, got %v", err)
	}

	_, err = NewTokenManager("secret", 0)
	if !errors.Is(err, ErrInvalidTokenTTL) {
		t.Fatalf("expected ErrInvalidTokenTTL, got %v", err)
	}
}

func TestTokenManagerGenerateReturnsErrorForEmptyUserID(t *testing.T) {
	manager, err := NewTokenManager("secret", time.Minute)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}

	_, err = manager.Generate("")
	if !errors.Is(err, ErrEmptyUserID) {
		t.Fatalf("expected ErrEmptyUserID, got %v", err)
	}
}

func TestTokenManagerValidateReturnsErrorForInvalidToken(t *testing.T) {
	manager, err := NewTokenManager("secret", time.Minute)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}

	_, err = manager.Validate("not-a-token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
