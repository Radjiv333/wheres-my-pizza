package repository

import (
	"context"
	"fmt"
	"strings"
	"time"
	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/internal/core/ports"
	"wheres-my-pizza/internal/core/services"
	"wheres-my-pizza/pkg/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Conn       *pgxpool.Pool
	DurationMs time.Duration
}

var _ ports.RepositoryInterface = (*Repository)(nil)

func NewRepository(cfg config.Config) (*Repository, error) {
	start := time.Now()
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

	durationMs := time.Since(start).Milliseconds()
	return &Repository{Conn: conn, DurationMs: time.Duration(durationMs)}, nil
}

// ORDERS
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
	order.Number, err = services.GenerateOrderNumber(ctx, tx)
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
	order.Status = "received"
	err = tx.QueryRow(ctx, insertOrderSQL,
		order.Number,
		order.CustomerName,
		order.Type,
		order.TableNumber,
		order.DeliveryAddress,
		order.TotalAmount,
		order.Priority,
		order.Status,      // initial status
		order.ProcessedBy, // can be null
		order.CompletedAt, // can be null
	).Scan(&order.ID)
	if err != nil {
		return "", err
	}

	// Insert order items
	const insertItemSQL = `
		INSERT INTO order_items (order_id, name, quantity, price)
		VALUES ($1, $2, $3, $4);
	`
	for _, item := range order.Items {
		if _, err := tx.Exec(ctx, insertItemSQL, order.ID, item.Name, item.Quantity, item.Price); err != nil {
			return "", err
		}
	}

	// Insert into order_status_log
	const insertStatusLogSQL = `
		INSERT INTO order_status_log (order_id, status, changed_by, notes)
		VALUES ($1, $2, $3, $4);
	`
	if _, err := tx.Exec(ctx, insertStatusLogSQL, order.ID, "received", order.ProcessedBy, "Order created"); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return order.Number, nil
}

func (r *Repository) OrderIsCooking(ctx context.Context, workerName string, order *domain.Order) error {
	tx, err := r.Conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Step 1: Update orders table
	updateSQL := `
		update orders
		set status = 'cooking', processed_by = $1
		where id = $2
		`
	_, err = tx.Exec(ctx, updateSQL, workerName, order.ID)
	if err != nil {
		return err
	}

	// Step 2: Insert into order_status_log
	insertSQL := `
		insert into order_status_log (order_id, status, changed_by)
		values ($1, $2, $3)
		`
	_, err = tx.Exec(ctx, insertSQL, order.ID, "cooking", workerName)
	if err != nil {
		return err
	}

	// Commit both changes
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	order.ProcessedBy = &workerName
	order.Status = "cooking"
	return nil
}

func (r *Repository) GetOrderStatus(ctx context.Context, orderID int) (string, error) {
	const selectSQL = `
		SELECT status FROM orders WHERE id = $1;
	`
	var status string
	err := r.Conn.QueryRow(ctx, selectSQL, orderID).Scan(&status)
	if err != nil {
		return status, err
	}

	return status, nil
}

func (r *Repository) OrderIsReady(ctx context.Context, workerName string, order *domain.Order) error {
	tx, err := r.Conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Update order status → ready
	updateOrderSQL := `
        update orders
        set status = 'ready',
            completed_at = now()
        where id = $1
    `
	res, err := tx.Exec(ctx, updateOrderSQL, order.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("order %d not found", order.ID)
	}

	// Increment worker’s orders_processed count
	updateWorkerSQL := `
        update workers
        set orders_processed = orders_processed + 1,
            last_seen = now()
        where name = $1
    `
	res, err = tx.Exec(ctx, updateWorkerSQL, workerName)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("worker %s not found", workerName)
	}

	// Insert into order_status_log
	insertSQL := `
		insert into order_status_log (order_id, status, changed_by)
		values ($1, $2, $3)
	`
	_, err = tx.Exec(ctx, insertSQL, order.ID, "ready", workerName)
	if err != nil {
		return err
	}
	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	t := time.Now()
	order.CompletedAt = &t
	order.ProcessedBy = &workerName
	order.Status = "ready"

	return nil
}

// KITCHEN WORKERS
func (r *Repository) InsertWorker(ctx context.Context, workerName string, orderTypes []string) error {
	const insertSQL = `
		INSERT INTO workers (name, type, status, last_seen)
		VALUES ($1, $2, 'online', $3);
	`
	orderTypesStr := strings.Join(orderTypes, ",")
	_, err := r.Conn.Exec(ctx, insertSQL, workerName, orderTypesStr, time.Now().UTC())
	return err
}

func (r *Repository) UpdateWorkerStatus(ctx context.Context, workerName, status string) error {
	const updateSQL = `
		UPDATE workers
		SET status = $1, last_seen = $2
		WHERE name = $3;
	`
	_, err := r.Conn.Exec(ctx, updateSQL, status, time.Now().UTC(), workerName)
	return err
}

func (r *Repository) GetWorkerStatus(ctx context.Context, workerName string) (string, error) {
	const selectSQL = `
		SELECT status FROM workers WHERE name = $1;
	`
	var status string
	err := r.Conn.QueryRow(ctx, selectSQL, workerName).Scan(&status)
	if err != nil {
		return status, err
	}

	return status, nil
}

func (r *Repository) UpdateWorkerHeartbeat(ctx context.Context, workerName string) error {
	const updateSQL = `
		update workers
		set last_seen = now(),
			status = 'online'
		where name = $1
	`
	_, err := r.Conn.Exec(ctx, updateSQL, workerName)
	return err
}

// TRACKING SERVICE

func (r *Repository) GetOrderDetails(ctx context.Context, orderNumber string) (domain.OrderDetailsResponse, error) {
	const q = `
		SELECT number, status, completed_at, processed_by, updated_at
		FROM orders
		WHERE number = $1
	`
	orderDetails := domain.OrderDetailsResponse{}
	err := r.Conn.QueryRow(ctx, q, orderNumber).Scan(&orderDetails.OrderNumber, &orderDetails.CurrentStatus, &orderDetails.EstimatedCompletion, &orderDetails.ProcessedBy, &orderDetails.UpdatedAt)

	return orderDetails, err
}

func (r *Repository) GetOrderHistory(ctx context.Context, orderNumber string) ([]map[string]interface{}, error) {
	const q = `
		SELECT 
			osl.status, 
			COALESCE(osl.changed_by, '') AS changed_by, 
			osl.changed_at
		FROM order_status_log osl
		JOIN orders o ON o.id = osl.order_id
		WHERE o.number = $1
		ORDER BY osl.changed_at ASC;
	`
	rows, err := r.Conn.Query(ctx, q, orderNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var status, changedBy string
		var ts time.Time
		if err := rows.Scan(&status, &changedBy, &ts); err != nil {
			return nil, err
		}
		history = append(history, map[string]interface{}{
			"status":     status,
			"timestamp":  ts.UTC(),
			"changed_by": changedBy,
		})
	}

	return history, err
}

func (r *Repository) GetWorkersStatuses(ctx context.Context, heartbeatTimeout time.Duration) ([]map[string]interface{}, error) {
	const q = `
		SELECT name, status, orders_processed, last_seen
		FROM workers
	`
	rows, err := r.Conn.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []map[string]interface{}
	now := time.Now().UTC()

	for rows.Next() {
		var name, status string
		var ordersProcessed int
		var lastSeen time.Time
		if err := rows.Scan(&name, &status, &ordersProcessed, &lastSeen); err != nil {
			return nil, err
		}

		// Check offline threshold
		if now.Sub(lastSeen) > heartbeatTimeout*time.Second {
			status = "offline"
		}

		workers = append(workers, map[string]interface{}{
			"worker_name":      name,
			"status":           status,
			"orders_processed": ordersProcessed,
			"last_seen":        lastSeen.UTC(),
		})
	}

	return workers, err
}
