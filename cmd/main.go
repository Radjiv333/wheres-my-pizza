package main

import (
	"fmt"
	"net/http"

	"wheres-my-pizza/internal/adapters/handlers"
)

func main() {
	fmt.Println("Hello world!")
	
	http.HandleFunc("POST	/orders", handlers.PostOrder)
	http.ListenAndServe("localhost:8080", nil)
}
