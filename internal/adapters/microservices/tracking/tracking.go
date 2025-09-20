package tracking

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"wheres-my-pizza/internal/adapters/db/repository"
	"wheres-my-pizza/internal/core/services"
	"wheres-my-pizza/pkg/logger"

	"github.com/jackc/pgx/v5"
)

type TrackingService struct {
	port             int
	repo             *repository.Repository
	logger           *logger.Logger
	heartbeatTimeout time.Duration
}

func NewTrackingHandler(repo *repository.Repository, port int, logger *logger.Logger, heartbeatTimeout int) *TrackingService {
	hbt := time.Duration(heartbeatTimeout)
	return &TrackingService{port: port, repo: repo, logger: logger, heartbeatTimeout: hbt}
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
	services.WriteJSON(w, resp, http.StatusOK)
}

// GET /orders/{order_number}/history
func (t *TrackingService) GetOrderHistory(w http.ResponseWriter, r *http.Request) {
	orderNumber := r.PathValue("order_number")
	ctx := r.Context()

	history, err := t.repo.GetOrderHistory(ctx, orderNumber)
	if err != nil {
		http.Error(w, "could not get order history: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if len(history) == 0 {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	services.WriteJSON(w, history, http.StatusOK)
}

// GET /workers/status
func (t *TrackingService) GetWorkersStatuses(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workers, err := t.repo.GetWorkersStatuses(ctx, t.heartbeatTimeout)
	if err != nil {
		http.Error(w, "could not get workers statuses: "+err.Error(), http.StatusInternalServerError)
		return
	}

	services.WriteJSON(w, workers, http.StatusOK)
}

func (o *TrackingService) Stop(ctx context.Context, server *http.Server) {
	<-ctx.Done()
	o.repo.Conn.Close()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %+v", err)
	}
	log.Println("shutting down gracefully...")
}
