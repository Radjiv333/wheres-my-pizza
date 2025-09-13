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
	"wheres-my-pizza/internal/adapters/microservices/order"
	"wheres-my-pizza/internal/adapters/rabbitmq"

	"wheres-my-pizza/internal/core/services"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"
)

// {"timestamp":"2024-12-16T10:30:00.000Z","level":"INFO","service":"order-service","hostname":"order-service-789abc","request_id":"startup-001","action":"service_started","message":"Order Service started on port 3000","details":{"port":3000,"max_concurrent":50}}
// {"timestamp":"2024-12-16T10:30:01.000Z","level":"INFO","service":"order-service","hostname":"order-service-789abc","request_id":"startup-001","action":"db_connected","message":"Connected to PostgreSQL database","duration_ms":250}
// {"timestamp":"2024-12-16T10:30:02.000Z","level":"INFO","service":"order-service","hostname":"order-service-789abc","request_id":"startup-001","action":"rabbitmq_connected","message":"Connected to RabbitMQ exchange 'orders_topic'","duration_ms":150}
func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("cannot load the config properly: %v", err)
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
		fmt.Printf("cannot connect to db: %v", err)
		os.Exit(1)
	}
	logger.Info("", "db_connected", "Connected to PostgreSQL database", map[string]interface{}{"duration_ms": repo.DurationMs})

	// Initializing rabbitmq
	rabbit, err := rabbitmq.NewRabbitMq()
	if err != nil {
		fmt.Printf("cannot connect to rabbitmq: %v", err)
		os.Exit(1)
	}
	logger.Info("", "rabbitmq_connected", "Connected to RabbitMQ exchange "+"order_topic", map[string]interface{}{"duration_ms": rabbit.DurationMs})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initializing Order-service Handler
	orderHandler := order.NewOrderHandler(repo, rabbit, flags.MaxConcurrent, flags.Port)

	// Initializing Mux
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", orderHandler.PostOrder)
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", flags.Port),
		Handler: mux,
	}

	// Starting server
	go func() {
		logger.Info("", "service_started", "Order Service started on port"+server.Addr, map[string]interface{}{"details": map[string]interface{}{"port": flags.Port, "max_concurrent": flags.MaxConcurrent}})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Waiting for Ctrl+C signal
	<-ctx.Done()
	log.Println("shutting down gracefully...")
	repo.Conn.Close()
	rabbit.Ch.Close()
	rabbit.Conn.Close()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %+v", err)
	}
}
