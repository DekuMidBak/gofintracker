package gateway

import (
	"net/http"
	"time"

	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	UserID      string     `json:"user_id"`
	AccessToken string     `json:"access_token"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
}

func (h handler) register(w http.ResponseWriter, r *http.Request) {
	if h.clients.Users == nil {
		writeError(w, http.StatusInternalServerError, "service dependency is not configured")
		return
	}

	var req authRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.clients.Users.Register(r.Context(), &userv1.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	var createdAt *time.Time
	if resp.GetCreatedAt() != nil {
		value := resp.GetCreatedAt().AsTime()
		createdAt = &value
	}

	writeJSON(w, http.StatusCreated, authResponse{
		UserID:      resp.GetUserId(),
		AccessToken: resp.GetAccessToken(),
		CreatedAt:   createdAt,
	})
}

func (h handler) login(w http.ResponseWriter, r *http.Request) {
	if h.clients.Users == nil {
		writeError(w, http.StatusInternalServerError, "service dependency is not configured")
		return
	}

	var req authRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.clients.Users.Login(r.Context(), &userv1.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		UserID:      resp.GetUserId(),
		AccessToken: resp.GetAccessToken(),
	})
}
