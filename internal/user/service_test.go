package user

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DekuMidBak/gofintracker/internal/user/auth"
)

func TestServiceRegisterCreatesUserAndAccessToken(t *testing.T) {
	repository := newFakeRepository()
	tokens := &fakeTokenManager{}
	service := NewService(repository, tokens)

	result, err := service.Register(context.Background(), Credentials{
		Email:    " USER@Example.COM ",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if result.User.Email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", result.User.Email)
	}

	if result.User.PasswordHash == "secret" {
		t.Fatal("expected password to be hashed")
	}

	if result.AccessToken != "token:user-1" {
		t.Fatalf("expected access token, got %q", result.AccessToken)
	}
}

func TestServiceRegisterReturnsErrInvalidEmail(t *testing.T) {
	service := NewService(newFakeRepository(), &fakeTokenManager{})

	_, err := service.Register(context.Background(), Credentials{
		Email:    "not-email",
		Password: "secret",
	})
	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestServiceLoginReturnsAccessToken(t *testing.T) {
	repository := newFakeRepository()
	passwordHash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	repository.usersByEmail["user@example.com"] = User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	service := NewService(repository, &fakeTokenManager{})
	result, err := service.Login(context.Background(), Credentials{
		Email:    "USER@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if result.AccessToken != "token:user-1" {
		t.Fatalf("expected access token, got %q", result.AccessToken)
	}
}

func TestServiceLoginReturnsErrInvalidCredentialsForMissingUser(t *testing.T) {
	service := NewService(newFakeRepository(), &fakeTokenManager{})

	_, err := service.Login(context.Background(), Credentials{
		Email:    "missing@example.com",
		Password: "secret",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestServiceLoginReturnsErrInvalidCredentialsForWrongPassword(t *testing.T) {
	repository := newFakeRepository()
	passwordHash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	repository.usersByEmail["user@example.com"] = User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: passwordHash,
	}

	service := NewService(repository, &fakeTokenManager{})
	_, err = service.Login(context.Background(), Credentials{
		Email:    "user@example.com",
		Password: "wrong",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestServiceValidateToken(t *testing.T) {
	service := NewService(newFakeRepository(), &fakeTokenManager{})

	userID, err := service.ValidateToken("token:user-1")
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}

	if userID != "user-1" {
		t.Fatalf("expected user-1, got %q", userID)
	}
}

type fakeRepository struct {
	nextID       int
	usersByID    map[string]User
	usersByEmail map[string]User
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		nextID:       1,
		usersByID:    make(map[string]User),
		usersByEmail: make(map[string]User),
	}
}

func (r *fakeRepository) Create(_ context.Context, params CreateUserParams) (User, error) {
	if _, ok := r.usersByEmail[params.Email]; ok {
		return User{}, ErrEmailExists
	}

	now := time.Now()
	created := User{
		ID:           fmt.Sprintf("user-%d", r.nextID),
		Email:        params.Email,
		PasswordHash: params.PasswordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	r.nextID++
	r.usersByID[created.ID] = created
	r.usersByEmail[created.Email] = created

	return created, nil
}

func (r *fakeRepository) FindByID(_ context.Context, id string) (User, error) {
	found, ok := r.usersByID[id]
	if !ok {
		return User{}, ErrNotFound
	}

	return found, nil
}

func (r *fakeRepository) FindByEmail(_ context.Context, email string) (User, error) {
	found, ok := r.usersByEmail[email]
	if !ok {
		return User{}, ErrNotFound
	}

	return found, nil
}

type fakeTokenManager struct{}

func (m *fakeTokenManager) Generate(userID string) (string, error) {
	return "token:" + userID, nil
}

func (m *fakeTokenManager) Validate(accessToken string) (string, error) {
	if len(accessToken) <= len("token:") || accessToken[:len("token:")] != "token:" {
		return "", auth.ErrInvalidToken
	}

	return accessToken[len("token:"):], nil
}
