package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"wheres-my-pizza/internal/adapters/db/repository"
	"wheres-my-pizza/internal/adapters/microservices/kitchen"
	"wheres-my-pizza/internal/adapters/microservices/order"
	"wheres-my-pizza/internal/adapters/rabbitmq"

	"wheres-my-pizza/internal/core/services"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("cannot load the config properly: %v\n", err)
		os.Exit(1)
	}

	// Parsing flags
	flags, err := services.FlagParse()
	if err != nil {
		services.AppUsage()
		os.Exit(1)
	}

	logger := logger.NewLogger(flags.Mode)

	// Initializing repository
	repo, err := repository.NewRepository(*cfg)
	if err != nil {
		// Gracefull shutdown
		fmt.Printf("cannot connect to db: %v\n", err)
		os.Exit(1)
	}
	logger.Info("", "db_connected", "Connected to PostgreSQL database", map[string]interface{}{"duration_ms": repo.DurationMs})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch flags.Mode {
	case "order-service":
		// Initializing rabbitmq for orders
		orderRabbit, err := rabbitmq.NewOrderRabbit()
		if err != nil {
			// Gracefull shutdown
			fmt.Printf("cannot connect to rabbitmq: %v\n", err)
			os.Exit(1)
		}
		logger.Info("", "rabbitmq_connected", "Connected to RabbitMQ exchange "+"order_topic", map[string]interface{}{"duration_ms": orderRabbit.DurationMs})

		// Initializing Order-service
		orderService := order.NewOrderHandler(repo, orderRabbit, flags.Order.MaxConcurrent, flags.Order.Port, logger)

		// Initializing Mux
		mux := http.NewServeMux()
		mux.HandleFunc("POST /orders", orderService.PostOrder)
		server := http.Server{
			Addr:    fmt.Sprintf(":%d", flags.Order.Port),
			Handler: mux,
		}
		// Starting server
		go func() {
			logger.Info("", "service_started", "Order Service started on port"+server.Addr, map[string]interface{}{"details": map[string]interface{}{"port": flags.Order.Port, "max_concurrent": flags.Order.MaxConcurrent}})
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				stop()
				fmt.Printf("cannot start server: %v\n", err)
				os.Exit(1)
			}
		}()

		orderService.Stop(ctx, &server)
	case "kitchen-worker":
		// Initializing rabbitmq for kitchen
		kitchenRabbit, err := rabbitmq.NewKitchenRabbit(flags.Kitchen.OrderTypes, flags.Kitchen.WorkerName)
		if err != nil {
			// Gracefull shutdown
			fmt.Printf("cannot connect to rabbitmq: %v\n", err)
			os.Exit(1)
		}
		logger.Info("", "rabbitmq_connected", "Connected to RabbitMQ exchange "+"order_topic", map[string]interface{}{"duration_ms": kitchenRabbit.DurationMs})

		kitchenService := kitchen.NewKitchen(repo, kitchenRabbit, flags.Kitchen, logger)
		err = kitchenService.Start(ctx)
		if err != nil {
			// ERROR LOGGER -----------------------------------------------------
			fmt.Printf("cannot start kitchen-service: %v\n", err)
			stop()
			os.Exit(1)
		}

		kitchenService.Stop(ctx)
	}
}
