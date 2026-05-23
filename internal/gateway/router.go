package gateway

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type RouterConfig struct {
	Clients Clients
}

type handler struct {
	clients Clients
	logger  *slog.Logger
}

func NewRouter(logger *slog.Logger, config RouterConfig) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	handler := handler{
		clients: config.Clients,
		logger:  logger,
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(requestLogger(logger))
	router.Use(middleware.Recoverer)

	router.Get("/health", handler.health)
	router.Route("/api/v1/auth", func(router chi.Router) {
		router.Post("/register", handler.register)
		router.Post("/login", handler.login)
	})
	router.Group(func(router chi.Router) {
		router.Use(handler.requireAuth)
		router.Post("/api/v1/categories", handler.createCategory)
		router.Get("/api/v1/categories", handler.listCategories)
		router.Post("/api/v1/transactions", handler.createTransaction)
		router.Get("/api/v1/transactions", handler.listTransactions)
		router.Get("/api/v1/balance", handler.getBalance)
		router.Get("/api/v1/analytics/monthly", handler.getMonthlyAnalytics)
		router.Get("/api/v1/analytics/categories", handler.getCategoryAnalytics)
	})

	return router
}

func (h handler) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(recorder, r)

			logger.Info(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.Status(),
				"bytes", recorder.BytesWritten(),
				"duration", time.Since(startedAt).String(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
