package main

import (
	"log"
	"net/http"
	"os"

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

	orderHandler := order.NewOrderHandler(repo, flags.MaxConcurrent, flags.Port)

	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	mux.HandleFunc("POST	/orders", orderHandler.PostOrder)
	server.ListenAndServe()
}
