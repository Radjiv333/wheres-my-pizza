package ports

import (
	"context"

	"wheres-my-pizza/internal/core/domain"
)

type MessageBrokerInterface interface {
	// SetupRabbitMQ() error
	PublishOrderMessage(ctx context.Context, order domain.Order) error
}
