package order

import (
	"encoding/json"
	"fmt"
	"net/http"

	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/internal/core/ports"
	"wheres-my-pizza/internal/core/services"
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
	ctx := r.Context()
	var order domain.Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		// ERROR LOGGER
		http.Error(w, "Cannot decode the order", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Get a pooled connection for transactional work
	conn, err := o.repo.Conn.Acquire(ctx)
	if err != nil {
		http.Error(w, "DB unavailable", http.StatusInternalServerError)
		return
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		http.Error(w, "Cannot begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// Generate order number inside the transaction
	orderNumber, err := services.GenerateOrderNumber(ctx, tx)
	if err != nil {
		http.Error(w, "Cannot generate order number", http.StatusInternalServerError)
		return
	}

	// Example insert into your orders table
	const insertOrderSQL = `
		INSERT INTO orders (
			number, customer_name, type, table_number, delivery_address,
			total_amount, priority, status, processed_by, completed_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id;
	`
	var orderID int
	err = tx.QueryRow(ctx, insertOrderSQL,
		orderNumber,
		order.CustomerName,
		order.Type,
		order.TableNumber,
		order.DeliveryAddress,
		order.TotalAmount,
		order.Priority,
		"received",        // initial status
		order.ProcessedBy, // can be null
		order.CompletedAt, // can be null
	).Scan(&orderID)
	if err != nil {
		http.Error(w, "Cannot insert order", http.StatusInternalServerError)
		return
	}

	// Insert order items
	const insertItemSQL = `
		INSERT INTO order_items (order_id, name, quantity, price)
		VALUES ($1, $2, $3, $4);
	`
	for _, item := range order.Items {
		if _, err := tx.Exec(ctx, insertItemSQL, orderID, item.Name, item.Quantity, item.Price); err != nil {
			http.Error(w, "Cannot insert order item", http.StatusInternalServerError)
			return
		}
	}

	// Insert into order_status_log
	const insertStatusLogSQL = `
		INSERT INTO order_status_log (order_id, status, changed_by, notes)
		VALUES ($1, $2, $3, $4);
	`
	if _, err := tx.Exec(ctx, insertStatusLogSQL, orderID, "received", order.ProcessedBy, "Order created"); err != nil {
		http.Error(w, "Cannot insert status log", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "Commit failed", http.StatusInternalServerError)
		return
	}

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
