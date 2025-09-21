package ports

import (
	"context"
	"wheres-my-pizza/internal/core/domain"
)

type RepositoryInterface interface {
	InsertOrder(ctx context.Context, order *domain.Order) (string, error)
	InsertWorker(ctx context.Context, workerName string, orderTypes []string) error
	UpdateWorkerStatus(ctx context.Context, workerName, status string) error
	// Close(ctx context.Context, workerName string) error
}
