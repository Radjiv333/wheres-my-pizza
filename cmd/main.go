package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"wheres-my-pizza/internal/adapters/app"
	"wheres-my-pizza/internal/adapters/db/repository"

	"wheres-my-pizza/internal/core/services"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"
)

func main() {
	// Loading config
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("cannot load the config properly: %v\n", err)
		os.Exit(1)
	}

	// Parsing flags
	flags, err := services.FlagParse()
	if err != nil {
		fmt.Println(err)
		services.AppUsage()
		os.Exit(1)
	}

	logger := logger.NewLogger(flags.Mode)

	// Initializing repository
	repo, err := repository.NewRepository(*cfg)
	if err != nil {
		logger.Error("", "db_connection_failed", "Database is unreachable after all retries", err, nil)
		os.Exit(1)
	}
	logger.Info("", "db_connected", "Connected to PostgreSQL database", map[string]interface{}{"duration_ms": repo.DurationMs})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch flags.Mode {
	case "order-service":
		app.Order(ctx, logger, repo, flags, stop)
	case "kitchen-worker":
		app.Kitchen(ctx, logger, repo, flags, stop)
	case "tracking-service":
		app.Tracking(ctx, logger, repo, flags, stop)
	case "notification-subscriber":
		app.Notification(ctx, logger)
	}
}
