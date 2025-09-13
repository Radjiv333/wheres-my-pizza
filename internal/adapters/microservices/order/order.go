package order

import (
	"encoding/json"
	"fmt"
	"net/http"

	"wheres-my-pizza/internal/adapters/db/repository"
	"wheres-my-pizza/internal/adapters/rabbitmq"

	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/internal/core/ports"
	"wheres-my-pizza/internal/core/services"
)

type OrderService struct {
	maxConcurrent int
	port          int
	repo          *repository.Repository
	rabbit        *rabbitmq.Rabbit
}

var _ ports.OrderServiceInterface = (*OrderService)(nil)

func NewOrderHandler(repo *repository.Repository, rabbit *rabbitmq.Rabbit, maxConcurrent, port int) *OrderService {
	return &OrderService{maxConcurrent: maxConcurrent, rabbit: rabbit, port: port, repo: repo}
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

	err = services.CheckOrderValues(order)
	if err != nil {
		// ERROR LOGGER
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	orderNumber, err := o.repo.InsertOrder(ctx, &order)
	if err != nil {
		// ERROR LOGGER
		http.Error(w, "Cannot insert the order to db: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = o.rabbit.PublishOrderMessage(ctx, order)
	if err != nil {
		// ERROR LOGGER
		http.Error(w, "Cannot publish order message: "+err.Error(), http.StatusInternalServerError)
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
