package gateway

import (
	"context"
	"net/http"
	"strings"

	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

func (h handler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.clients.Users == nil {
			writeError(w, http.StatusInternalServerError, "service dependency is not configured")
			return
		}

		authorization := r.Header.Get("Authorization")
		if authorization == "" {
			writeError(w, http.StatusUnauthorized, "authorization header is required")
			return
		}

		scheme, token, ok := strings.Cut(authorization, " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			writeError(w, http.StatusUnauthorized, "invalid authorization header")
			return
		}

		resp, err := h.clients.Users.ValidateToken(r.Context(), &userv1.ValidateTokenRequest{
			AccessToken: strings.TrimSpace(token),
		})
		if err != nil {
			writeGRPCError(w, err)
			return
		}

		if resp.GetUserId() == "" {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, resp.GetUserId())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok && userID != ""
}
