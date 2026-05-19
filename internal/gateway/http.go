package gateway

import (
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type errorResponse struct {
	Error string `json:"error"`
}

func readJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, errorResponse{Error: message})
}

func writeGRPCError(w http.ResponseWriter, err error) {
	code := status.Code(err)
	message := status.Convert(err).Message()

	switch code {
	case codes.InvalidArgument:
		writeError(w, http.StatusBadRequest, message)
	case codes.Unauthenticated:
		writeError(w, http.StatusUnauthorized, message)
	case codes.AlreadyExists:
		writeError(w, http.StatusConflict, message)
	case codes.NotFound:
		writeError(w, http.StatusNotFound, message)
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
