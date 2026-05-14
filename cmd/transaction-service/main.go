package main

import (
	"context"
	"fmt"
	"net"
	"os"

	transactionv1 "github.com/DekuMidBak/gofintracker/gen/go/transaction/v1"
	"github.com/DekuMidBak/gofintracker/internal/app"
	"github.com/DekuMidBak/gofintracker/internal/config"
	"github.com/DekuMidBak/gofintracker/internal/logging"
	"github.com/DekuMidBak/gofintracker/internal/transaction"
	transactiongrpc "github.com/DekuMidBak/gofintracker/internal/transaction/grpc"
	transactionpostgres "github.com/DekuMidBak/gofintracker/internal/transaction/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type serviceConfig struct {
	GRPCAddr  string
	DBDSN     string
	LogLevel  string
	LogFormat string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "transaction-service failed: %v\n", err)
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

	pool, err := pgxpool.New(ctx, cfg.DBDSN)
	if err != nil {
		return fmt.Errorf("connect to transactions database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping transactions database: %w", err)
	}

	repository := transactionpostgres.New(pool)
	service := transaction.NewService(repository)
	server := grpc.NewServer()
	transactionv1.RegisterTransactionServiceServer(server, transactiongrpc.NewServer(service))

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.GRPCAddr, err)
	}

	logger.Info("starting transaction-service", "grpc_addr", cfg.GRPCAddr)
	if err := app.ServeGRPC(ctx, listener, server); err != nil {
		return fmt.Errorf("serve grpc: %w", err)
	}

	logger.Info("stopped transaction-service")
	return nil
}

func loadConfig() (serviceConfig, error) {
	dbDSN, err := config.RequiredString("TRANSACTIONS_DATABASE_DSN")
	if err != nil {
		return serviceConfig{}, err
	}

	return serviceConfig{
		GRPCAddr:  config.String("TRANSACTION_SERVICE_GRPC_ADDR", ":50052"),
		DBDSN:     dbDSN,
		LogLevel:  config.String("LOG_LEVEL", "info"),
		LogFormat: config.String("LOG_FORMAT", logging.FormatJSON),
	}, nil
}
