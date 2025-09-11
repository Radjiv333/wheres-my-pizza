package order

import (
	"encoding/json"
	"fmt"
	"net/http"

	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/internal/core/ports"
)

type OrderService struct {
	maxConcurrent int
	port          int
	repo          *domain.Repository
}

var _ ports.OrderServiceInterface = (*OrderService)(nil)

func NewOrderHandler(repo *domain.Repository, maxConcurrent, port int) *OrderService {
	return &OrderService{maxConcurrent: maxConcurrent, port: port, repo: repo}
}

func (o *OrderService) Stop() {
	fmt.Println("Stop function")
}

func (o *OrderService) PostOrder(w http.ResponseWriter, r *http.Request) {
	var order domain.Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		// ERROR LOGGER
		http.Error(w, "Cannot decode the order", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	response := domain.PutOrderResponse{
		OrderNumber: "",
		Status:      "received",
		TotalAmount: 234.234,
	}

	responseByte, err := json.Marshal(response)
	if err != nil {
		// ERROR LOGGER
		http.Error(w, "Cannot marshal the order", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseByte)
}
