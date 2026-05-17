package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/DekuMidBak/gofintracker/internal/app"
	"github.com/DekuMidBak/gofintracker/internal/config"
	"github.com/DekuMidBak/gofintracker/internal/gateway"
	"github.com/DekuMidBak/gofintracker/internal/logging"
)

type serviceConfig struct {
	HTTPAddr        string
	ShutdownTimeout time.Duration
	LogLevel        string
	LogFormat       string
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

	server := &http.Server{
		Handler:           gateway.NewRouter(logger),
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
		HTTPAddr:        config.String("API_GATEWAY_HTTP_ADDR", ":8080"),
		ShutdownTimeout: shutdownTimeout,
		LogLevel:        config.String("LOG_LEVEL", "info"),
		LogFormat:       config.String("LOG_FORMAT", logging.FormatJSON),
	}, nil
}
