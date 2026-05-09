package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
	"github.com/DekuMidBak/gofintracker/internal/app"
	"github.com/DekuMidBak/gofintracker/internal/config"
	"github.com/DekuMidBak/gofintracker/internal/logging"
	"github.com/DekuMidBak/gofintracker/internal/user"
	"github.com/DekuMidBak/gofintracker/internal/user/auth"
	usergrpc "github.com/DekuMidBak/gofintracker/internal/user/grpc"
	userpostgres "github.com/DekuMidBak/gofintracker/internal/user/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type serviceConfig struct {
	GRPCAddr  string
	DBDSN     string
	JWTSecret string
	JWTTTL    time.Duration
	LogLevel  string
	LogFormat string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "user-service failed: %v\n", err)
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
		return fmt.Errorf("connect to users database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping users database: %w", err)
	}

	tokenManager, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		return err
	}

	repository := userpostgres.New(pool)
	service := user.NewService(repository, tokenManager)
	server := grpc.NewServer()
	userv1.RegisterUserServiceServer(server, usergrpc.NewServer(service))

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.GRPCAddr, err)
	}

	logger.Info("starting user-service", "grpc_addr", cfg.GRPCAddr)
	if err := app.ServeGRPC(ctx, listener, server); err != nil {
		return fmt.Errorf("serve grpc: %w", err)
	}

	logger.Info("stopped user-service")
	return nil
}

func loadConfig() (serviceConfig, error) {
	dbDSN, err := config.RequiredString("USERS_DATABASE_DSN")
	if err != nil {
		return serviceConfig{}, err
	}

	jwtSecret, err := config.RequiredString("JWT_SECRET")
	if err != nil {
		return serviceConfig{}, err
	}

	jwtTTL, err := config.Duration("JWT_ACCESS_TTL", 15*time.Minute)
	if err != nil {
		return serviceConfig{}, err
	}

	return serviceConfig{
		GRPCAddr:  config.String("USER_SERVICE_GRPC_ADDR", ":50051"),
		DBDSN:     dbDSN,
		JWTSecret: jwtSecret,
		JWTTTL:    jwtTTL,
		LogLevel:  config.String("LOG_LEVEL", "info"),
		LogFormat: config.String("LOG_FORMAT", logging.FormatJSON),
	}, nil
}
