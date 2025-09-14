package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Initializing rabbitmq
	rabbit, err := rabbitmq.NewRabbitMq(flags.Mode)
	if err != nil {
		// Gracefull shutdown
		fmt.Printf("cannot connect to rabbitmq: %v\n", err)
		os.Exit(1)
	}
	logger.Info("", "rabbitmq_connected", "Connected to RabbitMQ exchange "+"order_topic", map[string]interface{}{"duration_ms": rabbit.DurationMs})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var orderService *order.OrderService
	var kitchenService *kitchen.KitchenService
	var server http.Server
	switch flags.Mode {
	case "order-service":
		// Initializing Order-service
		orderService = order.NewOrderHandler(repo, rabbit, flags.Order.MaxConcurrent, flags.Order.Port, logger)

		// Initializing Mux
		mux := http.NewServeMux()
		mux.HandleFunc("POST /orders", orderService.PostOrder)
		server = http.Server{
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
	case "kitchen-worker":
		kitchenService = kitchen.NewKitchen(repo, rabbit, flags.Kitchen, logger)
		err := kitchenService.Start(ctx)
		if err != nil {
			// ERROR LOGGER -----------------------------------------------------
			fmt.Printf("cannot start kitchen-service: %v\n", err)
			stop()
			os.Exit(1)
		}

	}

	// Waiting for Ctrl+C signal
	<-ctx.Done()
	err = repo.UpdateWorkerStatus(context.Background(), flags.Kitchen.WorkerName, "offline")
	if err != nil {
		fmt.Printf("db cannot gracefully shutdown: %v\n", err)
	}
	repo.Conn.Close()
	rabbit.Ch.Close()
	rabbit.Conn.Close()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %+v", err)
	}
	log.Println("shutting down gracefully...")
}
