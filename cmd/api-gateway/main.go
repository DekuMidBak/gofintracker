package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	analyticsv1 "github.com/DekuMidBak/gofintracker/gen/go/analytics/v1"
	transactionv1 "github.com/DekuMidBak/gofintracker/gen/go/transaction/v1"
	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
	"github.com/DekuMidBak/gofintracker/internal/app"
	"github.com/DekuMidBak/gofintracker/internal/config"
	"github.com/DekuMidBak/gofintracker/internal/gateway"
	"github.com/DekuMidBak/gofintracker/internal/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type serviceConfig struct {
	HTTPAddr               string
	ShutdownTimeout        time.Duration
	UserServiceAddr        string
	TransactionServiceAddr string
	AnalyticsServiceAddr   string
	LogLevel               string
	LogFormat              string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "api-gateway failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	logger, err := logging.New(os.Stdout, logging.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})
	if err != nil {
		return err
	}

	ctx, cancel := app.ShutdownContext(context.Background())
	defer cancel()

	userConn, err := grpc.NewClient(cfg.UserServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create user-service grpc client: %w", err)
	}
	defer closeGRPCConn(logger, "user-service", userConn)

	transactionConn, err := grpc.NewClient(cfg.TransactionServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create transaction-service grpc client: %w", err)
	}
	defer closeGRPCConn(logger, "transaction-service", transactionConn)

	analyticsConn, err := grpc.NewClient(cfg.AnalyticsServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create analytics-service grpc client: %w", err)
	}
	defer closeGRPCConn(logger, "analytics-service", analyticsConn)

	clients := gateway.Clients{
		Users:        userv1.NewUserServiceClient(userConn),
		Transactions: transactionv1.NewTransactionServiceClient(transactionConn),
		Analytics:    analyticsv1.NewAnalyticsServiceClient(analyticsConn),
	}

	server := &http.Server{
		Handler: gateway.NewRouter(logger, gateway.RouterConfig{
			Clients: clients,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.HTTPAddr, err)
	}

	logger.Info("starting api-gateway", "http_addr", cfg.HTTPAddr)
	if err := app.ServeHTTP(ctx, listener, server, cfg.ShutdownTimeout); err != nil {
		return fmt.Errorf("serve http: %w", err)
	}

	logger.Info("stopped api-gateway")
	return nil
}

func loadConfig() (serviceConfig, error) {
	shutdownTimeout, err := config.Duration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return serviceConfig{}, err
	}

	return serviceConfig{
		HTTPAddr:               config.String("API_GATEWAY_HTTP_ADDR", ":8080"),
		ShutdownTimeout:        shutdownTimeout,
		UserServiceAddr:        config.String("API_GATEWAY_USER_SERVICE_GRPC_ADDR", "localhost:50051"),
		TransactionServiceAddr: config.String("API_GATEWAY_TRANSACTION_SERVICE_GRPC_ADDR", "localhost:50052"),
		AnalyticsServiceAddr:   config.String("API_GATEWAY_ANALYTICS_SERVICE_GRPC_ADDR", "localhost:50053"),
		LogLevel:               config.String("LOG_LEVEL", "info"),
		LogFormat:              config.String("LOG_FORMAT", logging.FormatJSON),
	}, nil
}

func closeGRPCConn(logger *slog.Logger, serviceName string, conn *grpc.ClientConn) {
	if err := conn.Close(); err != nil {
		logger.Warn("failed to close grpc client connection", "service", serviceName, "error", err)
	}
}
