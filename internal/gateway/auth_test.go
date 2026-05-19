package gateway

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRegister(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	users := &fakeUserClient{
		registerResponse: &userv1.RegisterResponse{
			UserId:      "user-1",
			AccessToken: "access-token",
			CreatedAt:   timestamppb.New(createdAt),
		},
	}
	router := newTestRouter(users)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		strings.NewReader(`{"email":"user@example.com","password":"secret"}`),
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	if users.registerRequest.GetEmail() != "user@example.com" {
		t.Fatalf("expected email to be forwarded")
	}

	var body authResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.UserID != "user-1" || body.AccessToken != "access-token" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestLogin(t *testing.T) {
	users := &fakeUserClient{
		loginResponse: &userv1.LoginResponse{
			UserId:      "user-1",
			AccessToken: "access-token",
		},
	}
	router := newTestRouter(users)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(`{"email":"user@example.com","password":"secret"}`),
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if users.loginRequest.GetPassword() != "secret" {
		t.Fatalf("expected password to be forwarded")
	}
}

func TestAuthReturnsBadRequestForInvalidJSON(t *testing.T) {
	router := newTestRouter(&fakeUserClient{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestAuthMapsGRPCErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
	}{
		{
			name:       "invalid argument",
			err:        status.Error(codes.InvalidArgument, "invalid email"),
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "unauthenticated",
			err:        status.Error(codes.Unauthenticated, "invalid credentials"),
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "already exists",
			err:        status.Error(codes.AlreadyExists, "email exists"),
			statusCode: http.StatusConflict,
		},
		{
			name:       "unknown",
			err:        status.Error(codes.Internal, "boom"),
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTestRouter(&fakeUserClient{
				loginErr: tt.err,
			})

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/auth/login",
				strings.NewReader(`{"email":"user@example.com","password":"secret"}`),
			)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.statusCode {
				t.Fatalf("expected status %d, got %d", tt.statusCode, rec.Code)
			}
		})
	}
}

func newTestRouter(users userv1.UserServiceClient) http.Handler {
	return NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterConfig{
		Clients: Clients{
			Users: users,
		},
	})
}

type fakeUserClient struct {
	registerRequest  *userv1.RegisterRequest
	registerResponse *userv1.RegisterResponse
	registerErr      error

	loginRequest  *userv1.LoginRequest
	loginResponse *userv1.LoginResponse
	loginErr      error
}

func (c *fakeUserClient) Register(
	_ context.Context,
	in *userv1.RegisterRequest,
	_ ...grpc.CallOption,
) (*userv1.RegisterResponse, error) {
	c.registerRequest = in
	if c.registerErr != nil {
		return nil, c.registerErr
	}

	return c.registerResponse, nil
}

func (c *fakeUserClient) Login(
	_ context.Context,
	in *userv1.LoginRequest,
	_ ...grpc.CallOption,
) (*userv1.LoginResponse, error) {
	c.loginRequest = in
	if c.loginErr != nil {
		return nil, c.loginErr
	}

	return c.loginResponse, nil
}

func (c *fakeUserClient) ValidateToken(
	context.Context,
	*userv1.ValidateTokenRequest,
	...grpc.CallOption,
) (*userv1.ValidateTokenResponse, error) {
	return &userv1.ValidateTokenResponse{}, nil
}
