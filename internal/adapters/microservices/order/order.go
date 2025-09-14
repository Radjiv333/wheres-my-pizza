package order

import (
	"encoding/json"
	"fmt"
	"net/http"

	"wheres-my-pizza/internal/adapters/db/repository"
	"wheres-my-pizza/internal/adapters/rabbitmq"
	"wheres-my-pizza/pkg/logger"

	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/internal/core/ports"
	"wheres-my-pizza/internal/core/services"
)

type OrderService struct {
	maxConcurrent int
	port          int
	repo          *repository.Repository
	rabbit        *rabbitmq.OrderRabbit
	logger        *logger.Logger
}

var _ ports.OrderServiceInterface = (*OrderService)(nil)

func NewOrderHandler(repo *repository.Repository, rabbit *rabbitmq.OrderRabbit, maxConcurrent, port int, logger *logger.Logger) *OrderService {
	return &OrderService{maxConcurrent: maxConcurrent, rabbit: rabbit, port: port, repo: repo, logger: logger}
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
		o.logger.Error("", "validation_failed", "The order data failed validation step", err, nil)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	o.logger.Debug("", "order_received", "New valid order is received", nil)

	orderNumber, err := o.repo.InsertOrder(ctx, &order)
	if err != nil {
		o.logger.Error("", "db_transaction_failed", "The transaction of order data into db is failed", err, nil)
		http.Error(w, "Cannot insert the order to db: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = o.rabbit.PublishOrderMessage(ctx, order)
	if err != nil {
		o.logger.Error("", "rabbitmq_publish_failed", "The publishing of the order message failed.", err, nil)
		http.Error(w, "Cannot publish order message: "+err.Error(), http.StatusInternalServerError)
		return
	}
	o.logger.Debug("", "order_published", "The order is successfully published to RabbitMQ", nil)

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
