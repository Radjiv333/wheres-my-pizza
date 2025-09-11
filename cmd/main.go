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
	order "wheres-my-pizza/internal/adapters/microservices/orders"

	"wheres-my-pizza/internal/core/services"
	"wheres-my-pizza/pkg/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		// ERROR LOGGER
		log.Fatalf("wrong config: %v", err)
	}
	repo, err := repository.NewRepository(*cfg)
	if err != nil {
		// ERROR LOGGER
		log.Fatalf("cannot connect to db: %v", err)
	}

	flags, err := services.FlagParse()
	if err != nil {
		// ERROR LOGGER
		services.AppUsage()
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	orderHandler := order.NewOrderHandler(repo, flags.MaxConcurrent, flags.Port)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", orderHandler.PostOrder)

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", flags.Port),
		Handler: mux,
	}

	go func() {
		log.Printf("server listening on %s\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	server.ListenAndServe()

	<-ctx.Done()
	log.Println("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %+v", err)
	}
}
