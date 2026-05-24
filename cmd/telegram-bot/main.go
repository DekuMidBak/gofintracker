package main

import (
	"context"
	"fmt"
	"os"

	"github.com/DekuMidBak/gofintracker/internal/app"
	"github.com/DekuMidBak/gofintracker/internal/config"
	"github.com/DekuMidBak/gofintracker/internal/logging"
	"github.com/DekuMidBak/gofintracker/internal/telegrambot"
)

type serviceConfig struct {
	BotToken       string
	GatewayBaseURL string
	LogLevel       string
	LogFormat      string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "telegram-bot failed: %v\n", err)
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

	logger.Info(
		"starting telegram-bot",
		"gateway_base_url", cfg.GatewayBaseURL,
		"bot_token_configured", cfg.BotToken != "",
	)

	telegramClient := telegrambot.NewTelegramClient(cfg.BotToken, nil)
	bot := telegrambot.NewBot(telegramClient, logger)
	if err := bot.Run(ctx); err != nil {
		return fmt.Errorf("run telegram bot: %w", err)
	}

	logger.Info("stopped telegram-bot")
	return nil
}

func loadConfig() (serviceConfig, error) {
	botToken, err := config.RequiredString("TELEGRAM_BOT_TOKEN")
	if err != nil {
		return serviceConfig{}, err
	}

	return serviceConfig{
		BotToken:       botToken,
		GatewayBaseURL: config.String("TELEGRAM_GATEWAY_BASE_URL", "http://localhost:8080"),
		LogLevel:       config.String("LOG_LEVEL", "info"),
		LogFormat:      config.String("LOG_FORMAT", logging.FormatJSON),
	}, nil
}
