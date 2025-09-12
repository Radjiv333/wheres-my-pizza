package repository

import (
	"context"
	"fmt"

	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/internal/core/services"
	"wheres-my-pizza/pkg/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Conn *pgxpool.Pool
}

// "postgres://username:password@localhost:5432/database_name"
func NewRepository(cfg config.Config) (*Repository, error) {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.DatabaseName)

	conn, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return &Repository{}, err
	}

	var greeting string
	err = conn.QueryRow(context.Background(), "select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		return nil, err
	}

	fmt.Println(greeting)

	return &Repository{Conn: conn}, nil
}

func (r *Repository) InsertOrder(ctx context.Context, order *domain.Order) (string, error) {
	// Get a pooled connection for transactional work
	conn, err := r.Conn.Acquire(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	// Generate order number inside the transaction
	orderNumber, err := services.GenerateOrderNumber(ctx, tx)
	if err != nil {
		return "", err
	}

	// Calculating order's total price/amount
	order.TotalAmount = 0
	for _, item := range order.Items {
		order.TotalAmount += float64(item.Quantity) * item.Price
	}

	// Calculating order's priority based on total_amount
	order.Priority = services.AssignPriority(*order)

	// Example insert into orders table
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
		return "", err
	}

	// Insert order items
	const insertItemSQL = `
		INSERT INTO order_items (order_id, name, quantity, price)
		VALUES ($1, $2, $3, $4);
	`
	for _, item := range order.Items {
		if _, err := tx.Exec(ctx, insertItemSQL, orderID, item.Name, item.Quantity, item.Price); err != nil {
			return "", err
		}
	}

	// Insert into order_status_log
	const insertStatusLogSQL = `
		INSERT INTO order_status_log (order_id, status, changed_by, notes)
		VALUES ($1, $2, $3, $4);
	`
	if _, err := tx.Exec(ctx, insertStatusLogSQL, orderID, "received", order.ProcessedBy, "Order created"); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return orderNumber, nil
}
