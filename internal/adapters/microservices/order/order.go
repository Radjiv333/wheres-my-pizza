package order

import (
	"encoding/json"
	"fmt"
	"net/http"

	"wheres-my-pizza/internal/adapters/db/repository"
	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/internal/core/ports"
)

type OrderService struct {
	maxConcurrent int
	port          int
	repo          *repository.Repository
}

var _ ports.OrderServiceInterface = (*OrderService)(nil)

func NewOrderHandler(repo *repository.Repository, maxConcurrent, port int) *OrderService {
	return &OrderService{maxConcurrent: maxConcurrent, port: port, repo: repo}
}

func (o *OrderService) Stop() {
	fmt.Println("Stop function")
}

func (o *OrderService) PostOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var order domain.Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		// ERROR LOGGER
		http.Error(w, "Cannot decode the order", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	orderNumber, err := o.repo.InsertOrder(ctx, &order)
	if err != nil {
		http.Error(w, "Cannot insert the order to db: "+err.Error(), http.StatusBadRequest)
		return
	}

	response := domain.PutOrderResponse{
		OrderNumber: orderNumber,
		Status:      "received",
		TotalAmount: order.TotalAmount,
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
