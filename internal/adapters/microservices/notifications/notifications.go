package notifications

import (
	"context"
	"log"

	"wheres-my-pizza/internal/adapters/rabbitmq"
	"wheres-my-pizza/pkg/logger"
)

type NotificationService struct {
	logger *logger.Logger
	rabbit *rabbitmq.NotificationRabbit
}

func NewNotificationService(rabbit *rabbitmq.NotificationRabbit, logger *logger.Logger) *NotificationService {
	return &NotificationService{rabbit: rabbit, logger: logger}
}

func (n *NotificationService) Start(ctx context.Context) error {
	err := n.rabbit.ConsumeMessages(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (n *NotificationService) Stop(ctx context.Context) {
	<-ctx.Done()

	n.rabbit.Ch.Close()
	n.rabbit.Conn.Close()
	log.Println("shutting down gracefully...")
}
