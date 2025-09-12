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

	"wheres-my-pizza/internal/core/services"
	"wheres-my-pizza/pkg/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		// ERROR LOGGER
		log.Fatalf("wrong config: %v", err)
	}
	// INFO LOGGER

	// Initializing repository
	repo, err := repository.NewRepository(*cfg)
	if err != nil {
		// ERROR LOGGER
		log.Fatalf("cannot connect to db: %v", err)
	}
	// INFO LOGGER

	

	// Parsing flags
	flags, err := services.FlagParse()
	if err != nil {
		// ERROR LOGGER
		services.AppUsage()
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initializing Order-service Handler
	orderHandler := order.NewOrderHandler(repo, flags.MaxConcurrent, flags.Port)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", orderHandler.PostOrder)
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", flags.Port),
		Handler: mux,
	}

	// Starting server
	go func() {
		// INFO LOGGER
		log.Printf("server listening on %s\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Waiting for Ctrl+C signal
	<-ctx.Done()
	log.Println("shutting down gracefully...")
	repo.Conn.Close()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %+v", err)
	}
}
