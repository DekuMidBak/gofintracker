package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	analyticsv1 "github.com/DekuMidBak/gofintracker/gen/go/analytics/v1"
	"github.com/DekuMidBak/gofintracker/internal/analytics"
	analyticsgrpc "github.com/DekuMidBak/gofintracker/internal/analytics/grpc"
	analyticskafka "github.com/DekuMidBak/gofintracker/internal/analytics/kafka"
	analyticspostgres "github.com/DekuMidBak/gofintracker/internal/analytics/postgres"
	"github.com/DekuMidBak/gofintracker/internal/app"
	"github.com/DekuMidBak/gofintracker/internal/config"
	"github.com/DekuMidBak/gofintracker/internal/logging"
	"github.com/DekuMidBak/gofintracker/internal/transaction"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type serviceConfig struct {
	GRPCAddr     string
	DBDSN        string
	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroupID string
	LogLevel     string
	LogFormat    string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "analytics-service failed: %v\n", err)
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
		return fmt.Errorf("connect to analytics database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping analytics database: %w", err)
	}

	repository := analyticspostgres.New(pool)
	service := analytics.NewService(repository)
	server := grpc.NewServer()
	analyticsv1.RegisterAnalyticsServiceServer(server, analyticsgrpc.NewServer(service))

	consumer := analyticskafka.NewConsumer(analyticskafka.Config{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.KafkaTopic,
		GroupID: cfg.KafkaGroupID,
	}, service, logger)
	defer func() {
		if err := consumer.Close(); err != nil {
			logger.Warn("failed to close analytics event consumer", "error", err)
		}
	}()

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.GRPCAddr, err)
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		logger.Info("starting analytics-service grpc server", "grpc_addr", cfg.GRPCAddr)
		if err := app.ServeGRPC(groupCtx, listener, server); err != nil {
			return fmt.Errorf("serve grpc: %w", err)
		}

		return nil
	})

	group.Go(func() error {
		logger.Info(
			"starting analytics-service kafka consumer",
			"brokers", cfg.KafkaBrokers,
			"topic", cfg.KafkaTopic,
			"group_id", cfg.KafkaGroupID,
		)
		if err := consumer.Run(groupCtx); err != nil {
			return fmt.Errorf("run kafka consumer: %w", err)
		}

		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}

	logger.Info("stopped analytics-service")
	return nil
}

func loadConfig() (serviceConfig, error) {
	dbDSN, err := config.RequiredString("ANALYTICS_DATABASE_DSN")
	if err != nil {
		return serviceConfig{}, err
	}

	return serviceConfig{
		GRPCAddr:     config.String("ANALYTICS_SERVICE_GRPC_ADDR", ":50053"),
		DBDSN:        dbDSN,
		KafkaBrokers: splitCSV(config.String("KAFKA_BROKERS", "localhost:9094")),
		KafkaTopic:   config.String("KAFKA_TRANSACTION_CREATED_TOPIC", transaction.TransactionCreatedEventName),
		KafkaGroupID: config.String("ANALYTICS_KAFKA_GROUP_ID", "analytics-service"),
		LogLevel:     config.String("LOG_LEVEL", "info"),
		LogFormat:    config.String("LOG_FORMAT", logging.FormatJSON),
	}, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
