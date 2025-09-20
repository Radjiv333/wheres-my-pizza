package kitchen

import (
	"context"
	"fmt"
	"time"

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

	errCh := make(chan error)
	ch, err := k.rabbit.ConsumeMessages(ctx, k.kitchenFlags.WorkerName, errCh)
	if err != nil {
		return err
	}

	go k.getOrder(ctx, ch, errCh)
	return nil
}

func (k *KitchenService) getOrder(ctx context.Context, orderCh <-chan domain.Order, errCh chan error) {
	fmt.Println("in getOrder")
	for {
		select {
		case order := <-orderCh:
			err := k.repo.OrderIsCooking(ctx, k.kitchenFlags.WorkerName, "cooking", order.ID)
			if err != nil {
				errCh <- err
			}
			newOrderStatus, err := k.repo.GetOrderStatus(ctx, order.ID)
			if err != nil {
				errCh <- err
			}

			var cookingTime int
			switch order.Type {
			case "dine_in":
				cookingTime = 8
			case "takeout":
				cookingTime = 10
			case "delivery":
				cookingTime = 12
			}

			err = k.rabbit.PublishStatusUpdateMessage(ctx, order, newOrderStatus, k.kitchenFlags.WorkerName, cookingTime)
			if err != nil {
				errCh <- err
			}

			// Simulating work of workers
			k.simulateWork(ctx, cookingTime)

			err = k.repo.OrderIsReady(ctx, order.ID, k.kitchenFlags.WorkerName)
			if err != nil {
				errCh <- err
			}
		case <-ctx.Done():
			return
		}
	}
}

func (k *KitchenService) simulateWork(ctx context.Context, cookingTime int) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	counter := 0
	fmt.Print("cooking")
Loop:
	for {
		select {
		case <-ticker.C:
			if counter == cookingTime {
				fmt.Println()
				break Loop
			}
			counter++
			fmt.Print(".")
		case <-ctx.Done():
			return
		}
	}
	fmt.Println("finished cooking!")
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
