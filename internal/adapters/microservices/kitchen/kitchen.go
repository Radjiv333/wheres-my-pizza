package kitchen

import (
	"context"
	"fmt"

	"wheres-my-pizza/internal/adapters/db/repository"
	"wheres-my-pizza/internal/adapters/rabbitmq"
	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/internal/core/ports"
	"wheres-my-pizza/internal/core/services"
	"wheres-my-pizza/pkg/logger"

	"github.com/jackc/pgx/v5"
)

type KitchenService struct {
	repo         *repository.Repository
	rabbit       *rabbitmq.KitchenRabbit
	kitchenFlags services.KitchenFlags
	logger       *logger.Logger
}

var _ ports.KitchenServiceInterface = (*KitchenService)(nil)

func NewKitchen(repo *repository.Repository, rabbit *rabbitmq.KitchenRabbit, kitchenFlags services.KitchenFlags, logger *logger.Logger) *KitchenService {
	return &KitchenService{repo: repo, rabbit: rabbit, kitchenFlags: kitchenFlags, logger: logger}
}

func (k *KitchenService) Start(ctx context.Context) error {
	status, err := k.repo.GetWorkerStatus(ctx, k.kitchenFlags.WorkerName)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	switch status {
	case "online":
		return fmt.Errorf("worker is already working")
	case "offline":
		err := k.repo.UpdateWorkerStatus(ctx, k.kitchenFlags.WorkerName, "online")
		if err != nil {
			return err
		}
	case "":
		err := k.repo.InsertWorker(ctx, k.kitchenFlags.WorkerName, k.kitchenFlags.OrderTypes)
		if err != nil {
			return err
		}
	}

	ch, err := k.rabbit.ConsumeMessages(ctx, k.kitchenFlags.WorkerName)
	if err != nil {
		return err
	}

	go getOrder(ctx, ch)
	return nil
}

func getOrder(ctx context.Context, ch <-chan domain.Order) {
	for {
		select {
		case order := <-ch:
			fmt.Println("hello world", order)
		case <-ctx.Done():
			return
		}
	}
}

func (k *KitchenService) Stop(ctx context.Context) {
	<-ctx.Done()
	err := k.repo.UpdateWorkerStatus(context.Background(), k.kitchenFlags.WorkerName, "offline")
	if err != nil {
		fmt.Printf("db cannot gracefully shutdown: %v\n", err)
	}
	k.repo.Conn.Close()

	fmt.Println("shutting down gracefully...")
}
