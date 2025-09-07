package main

import (
	"net/http"
	"os"

	"wheres-my-pizza/internal/adapters/handlers"
	"wheres-my-pizza/internal/core/services"
)

func main() {
	err := services.FlagParse()
	if err != nil {
		// ERROR LOGGER
		services.AppUsage()
		os.Exit(1)
	}

	http.HandleFunc("POST	/orders", handlers.PostOrder)
	http.ListenAndServe("localhost:8080", nil)
}
