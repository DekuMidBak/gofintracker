package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
	"github.com/DekuMidBak/gofintracker/internal/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServerRegister(t *testing.T) {
	service := &fakeService{
		registerResult: user.AuthResult{
			User: user.User{
				ID:        "user-1",
				CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			},
			AccessToken: "access-token",
		},
	}
	server := NewServer(service)

	resp, err := server.Register(context.Background(), &userv1.RegisterRequest{
		Email:    "user@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if service.registerParams.Email != "user@example.com" {
		t.Fatalf("expected email to be forwarded")
	}

	if resp.GetUserId() != "user-1" {
		t.Fatalf("expected user id, got %q", resp.GetUserId())
	}

	if resp.GetAccessToken() != "access-token" {
		t.Fatalf("expected access token, got %q", resp.GetAccessToken())
	}

	if resp.GetCreatedAt().AsTime() != service.registerResult.User.CreatedAt {
		t.Fatalf("expected created_at timestamp")
	}
}

func TestServerLoginMapsInvalidCredentials(t *testing.T) {
	server := NewServer(&fakeService{
		loginErr: user.ErrInvalidCredentials,
	})

	_, err := server.Login(context.Background(), &userv1.LoginRequest{
		Email:    "user@example.com",
		Password: "wrong",
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %s", status.Code(err))
	}
}

func TestServerRegisterMapsDuplicateEmail(t *testing.T) {
	server := NewServer(&fakeService{
		registerErr: user.ErrEmailExists,
	})

	_, err := server.Register(context.Background(), &userv1.RegisterRequest{
		Email:    "user@example.com",
		Password: "secret",
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected already exists, got %s", status.Code(err))
	}
}

func TestServerValidateToken(t *testing.T) {
	server := NewServer(&fakeService{
		validateUserID: "user-1",
	})

	resp, err := server.ValidateToken(context.Background(), &userv1.ValidateTokenRequest{
		AccessToken: "access-token",
	})
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}

	if resp.GetUserId() != "user-1" {
		t.Fatalf("expected user id, got %q", resp.GetUserId())
	}
}

func TestServerMapsUnknownErrorToInternal(t *testing.T) {
	server := NewServer(&fakeService{
		registerErr: errors.New("boom"),
	})

	_, err := server.Register(context.Background(), &userv1.RegisterRequest{})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal, got %s", status.Code(err))
	}
}

type fakeService struct {
	registerParams user.RegisterParams
	registerResult user.AuthResult
	registerErr    error

	loginParams user.LoginParams
	loginResult user.AuthResult
	loginErr    error

	validateToken  string
	validateUserID string
	validateErr    error
}

func (s *fakeService) Register(_ context.Context, params user.RegisterParams) (user.AuthResult, error) {
	s.registerParams = params
	if s.registerErr != nil {
		return user.AuthResult{}, s.registerErr
	}

	return s.registerResult, nil
}

func (s *fakeService) Login(_ context.Context, params user.LoginParams) (user.AuthResult, error) {
	s.loginParams = params
	if s.loginErr != nil {
		return user.AuthResult{}, s.loginErr
	}

	return s.loginResult, nil
}

func (s *fakeService) ValidateToken(accessToken string) (string, error) {
	s.validateToken = accessToken
	if s.validateErr != nil {
		return "", s.validateErr
	}

	return s.validateUserID, nil
}
