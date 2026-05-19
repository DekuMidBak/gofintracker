package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRequireAuthStoresUserIDInContext(t *testing.T) {
	users := &fakeUserClient{
		validateResponse: &userv1.ValidateTokenResponse{
			UserId: "user-1",
		},
	}
	handler := handler{
		clients: Clients{Users: users},
	}

	protected := handler.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected user id in context")
		}

		if userID != "user-1" {
			t.Fatalf("expected user-1, got %q", userID)
		}

		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}

	if users.validateRequest.GetAccessToken() != "access-token" {
		t.Fatalf("expected token to be forwarded")
	}
}

func TestRequireAuthRejectsMissingAuthorizationHeader(t *testing.T) {
	handler := handler{
		clients: Clients{Users: &fakeUserClient{}},
	}
	protected := handler.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestRequireAuthRejectsInvalidAuthorizationHeader(t *testing.T) {
	handler := handler{
		clients: Clients{Users: &fakeUserClient{}},
	}
	protected := handler.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token access-token")
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestRequireAuthMapsValidateTokenError(t *testing.T) {
	handler := handler{
		clients: Clients{
			Users: &fakeUserClient{
				validateErr: status.Error(codes.Unauthenticated, "invalid token"),
			},
		},
	}
	protected := handler.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestRequireAuthReturnsInternalErrorWithoutUserClient(t *testing.T) {
	handler := handler{}
	protected := handler.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}
