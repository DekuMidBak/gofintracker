package gateway

import (
	"net/http"
	"time"

	transactionv1 "github.com/DekuMidBak/gofintracker/gen/go/transaction/v1"
)

type categoryRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type categoryResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

func (h handler) createCategory(w http.ResponseWriter, r *http.Request) {
	if h.clients.Transactions == nil {
		writeError(w, http.StatusInternalServerError, "service dependency is not configured")
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	var req categoryRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	categoryType, ok := transactionTypeFromString(req.Type)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid category type")
		return
	}

	resp, err := h.clients.Transactions.CreateCategory(r.Context(), &transactionv1.CreateCategoryRequest{
		UserId: userID,
		Name:   req.Name,
		Type:   categoryType,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toCategoryResponse(resp.GetCategory()))
}

func (h handler) listCategories(w http.ResponseWriter, r *http.Request) {
	if h.clients.Transactions == nil {
		writeError(w, http.StatusInternalServerError, "service dependency is not configured")
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	req := &transactionv1.ListCategoriesRequest{
		UserId: userID,
	}

	if rawType := r.URL.Query().Get("type"); rawType != "" {
		categoryType, ok := transactionTypeFromString(rawType)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid category type")
			return
		}

		req.Type = &categoryType
	}

	resp, err := h.clients.Transactions.ListCategories(r.Context(), req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	categories := make([]categoryResponse, 0, len(resp.GetCategories()))
	for _, category := range resp.GetCategories() {
		categories = append(categories, toCategoryResponse(category))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"categories": categories,
	})
}

func transactionTypeFromString(value string) (transactionv1.TransactionType, bool) {
	switch value {
	case "income":
		return transactionv1.TransactionType_TRANSACTION_TYPE_INCOME, true
	case "expense":
		return transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE, true
	default:
		return transactionv1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED, false
	}
}

func transactionTypeToString(transactionType transactionv1.TransactionType) string {
	switch transactionType {
	case transactionv1.TransactionType_TRANSACTION_TYPE_INCOME:
		return "income"
	case transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE:
		return "expense"
	default:
		return ""
	}
}

func toCategoryResponse(category *transactionv1.Category) categoryResponse {
	var createdAt time.Time
	if category.GetCreatedAt() != nil {
		createdAt = category.GetCreatedAt().AsTime()
	}

	return categoryResponse{
		ID:        category.GetId(),
		UserID:    category.GetUserId(),
		Name:      category.GetName(),
		Type:      transactionTypeToString(category.GetType()),
		CreatedAt: createdAt,
	}
}
