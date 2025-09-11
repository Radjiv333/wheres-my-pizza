package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"wheres-my-pizza/internal/adapters/db/repository"
	"wheres-my-pizza/internal/adapters/handlers"
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
	fmt.Println(repo)
	err = services.FlagParse()
	if err != nil {
		// ERROR LOGGER
		services.AppUsage()
		os.Exit(1)
	}

	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	mux.HandleFunc("POST	/orders", handlers.PostOrder)
	server.ListenAndServe()
}
