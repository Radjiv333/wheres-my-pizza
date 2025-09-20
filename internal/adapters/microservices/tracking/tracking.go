package tracking

import (
	"encoding/json"
	"net/http"
	"time"

	"wheres-my-pizza/internal/adapters/db/repository"
	"wheres-my-pizza/pkg/logger"

	"github.com/jackc/pgx/v5"
)

type TrackingService struct {
	port   int
	repo   *repository.Repository
	logger *logger.Logger
}

func NewTrackingHandler(repo *repository.Repository, port int, logger *logger.Logger) *TrackingService {
	return &TrackingService{port: port, repo: repo, logger: logger}
}

func (t *TrackingService) GetOrderDetails(w http.ResponseWriter, r *http.Request) {
	orderNumber := r.PathValue("order_number")
	ctx := r.Context()
	orderDetails, err := t.repo.GetOrderDetails(ctx, orderNumber)
	if err == pgx.ErrNoRows {
		http.Error(w, "order was not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "could not get order details from db: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(orderDetails)
	if err != nil {
		http.Error(w, "Cannot marshal the order", http.StatusInternalServerError)
		return
	}
	writeJSON(w, resp, http.StatusOK)
}

// GET /orders/{order_number}/history
func (t *TrackingService) GetOrderHistory(w http.ResponseWriter, r *http.Request) {
	orderNumber := r.PathValue("order_number")
	ctx := r.Context()

	const q = `
		SELECT osl.status, osl.changed_by, osl.changed_at
		FROM order_status_log osl
		JOIN orders o ON o.id = osl.order_id
		WHERE o.number = $1
		ORDER BY osl.changed_at ASC
	`
	rows, err := t.db.Query(ctx, q, orderNumber)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var status, changedBy string
		var ts time.Time
		if err := rows.Scan(&status, &changedBy, &ts); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		history = append(history, map[string]interface{}{
			"status":     status,
			"timestamp":  ts.UTC(),
			"changed_by": changedBy,
		})
	}

	if len(history) == 0 {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	writeJSON(w, history, http.StatusOK)
}

// GET /workers/status
func (t *TrackingService) GetWorkersStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	const q = `
		SELECT name, status, orders_processed, last_seen
		FROM workers
	`
	rows, err := t.db.Query(ctx, q)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var workers []map[string]interface{}
	now := time.Now().UTC()

	for rows.Next() {
		var name, status string
		var ordersProcessed int
		var lastSeen time.Time
		if err := rows.Scan(&name, &status, &ordersProcessed, &lastSeen); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		// Check offline threshold
		if now.Sub(lastSeen) > t.heartbeatTimeout {
			status = "offline"
		}

		workers = append(workers, map[string]interface{}{
			"worker_name":      name,
			"status":           status,
			"orders_processed": ordersProcessed,
			"last_seen":        lastSeen.UTC(),
		})
	}

	writeJSON(w, workers, http.StatusOK)
}

// ---------- Helpers ----------

func writeJSON(w http.ResponseWriter, v interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
