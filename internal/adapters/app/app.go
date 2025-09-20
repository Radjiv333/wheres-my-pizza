package app

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"wheres-my-pizza/internal/adapters/db/repository"
	"wheres-my-pizza/internal/adapters/microservices/kitchen"
	"wheres-my-pizza/internal/adapters/microservices/order"
	"wheres-my-pizza/internal/adapters/microservices/tracking"
	"wheres-my-pizza/internal/adapters/rabbitmq"
	"wheres-my-pizza/internal/core/services"
	"wheres-my-pizza/pkg/logger"
)

func Order(ctx context.Context, logger *logger.Logger, repo *repository.Repository, flags services.Flags, stop context.CancelFunc) {
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
}

func Kitchen(ctx context.Context, logger *logger.Logger, repo *repository.Repository, flags services.Flags, stop context.CancelFunc) {
	// Initializing rabbitmq for kitchen
	kitchenRabbit, err := rabbitmq.NewKitchenRabbit(flags.Kitchen.OrderTypes, flags.Kitchen.WorkerName, flags.Kitchen.Prefetch, logger)
	if err != nil {
		// Gracefull shutdown
		fmt.Printf("cannot connect to rabbitmq: %v\n", err)
		os.Exit(1)
	}
	logger.Info("", "rabbitmq_connected", "Connected to RabbitMQ exchange "+"order_topic", map[string]interface{}{"duration_ms": kitchenRabbit.DurationMs})

	// Initializing Kitchen service
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

func Tracking(ctx context.Context, logger *logger.Logger, repo *repository.Repository, flags services.Flags, stop context.CancelFunc) {
	// Initializing Order-service
	trackingService := tracking.NewTrackingHandler(repo, flags.Order.Port, logger)

	// Initializing Mux
	trackingMUX := http.NewServeMux()

	trackingMUX.HandleFunc("GET /orders/{order_number}/status", trackingService.GetOrderDetails)
	trackingMUX.HandleFunc("GET /orders/{order_number}/history", trackingService.GetOrderHistory)
	trackingMUX.HandleFunc("GET /workers/status", trackingService.GetWorkersStatus)

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", flags.Order.Port),
		Handler: trackingMUX,
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
}
