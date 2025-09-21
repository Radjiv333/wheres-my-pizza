package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/pkg/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

type OrderRabbitInterface interface{}

var _ OrderRabbitInterface = (*OrderRabbit)(nil)

type OrderRabbit struct {
	Conn       *amqp.Connection
	Ch         *amqp.Channel
	DurationMs time.Duration
	url        string
	logger     *logger.Logger
}

func NewOrderRabbit() (*OrderRabbit, error) {
	r := &OrderRabbit{url: "amqp://guest:guest@localhost:5672/"}
	if err := r.connect(); err != nil {
		return nil, err
	}
	// Watch for close signals
	go r.handleReconnect(5 * time.Second)
	return r, nil
}

func (r *OrderRabbit) connect() error {
	start := time.Now()
	conn, err := amqp.Dial(r.url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	if err := SetupOrderChannel(ch); err != nil {
		conn.Close()
		return err
	}

	r.Conn = conn
	r.Ch = ch
	r.DurationMs = time.Since(start)

	return nil
}

func SetupOrderChannel(ch *amqp.Channel) error {
	// --- Declare Exchanges ---
	if err := ch.ExchangeDeclare(
		"orders_topic", // name
		"topic",        // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // args
	); err != nil {
		return err
	}

	return nil
}

func (r *OrderRabbit) handleReconnect(backoff time.Duration) {
	errs := make(chan *amqp.Error)
	r.Conn.NotifyClose(errs)
	for e := range errs {
		fmt.Printf("RabbitMQ connection closed: %v. Reconnecting...\n", e)
		// Retry with backoff
		for {
			time.Sleep(backoff)
			if err := r.connect(); err != nil {
				fmt.Printf("Reconnect failed: %v\n", err)
				continue
			}
			// Restart notify channel
			errs = make(chan *amqp.Error)
			r.Conn.NotifyClose(errs)
			fmt.Println("Reconnect is succefull")
			break
		}
	}
}

func (r *OrderRabbit) PublishOrderMessage(ctx context.Context, order domain.Order) error {
	body, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order message: %w", err)
	}

	// Create routing key: kitchen.{order_type}.{priority}
	routingKey := fmt.Sprintf("kitchen.%s.%d", order.Type, order.Priority)

	// Publish to exchange
	err = r.Ch.PublishWithContext(
		ctx,            // context
		"orders_topic", // exchange
		routingKey,     // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // make message persistent
			Priority:     uint8(order.Priority),
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *OrderRabbit) Close() {
	if r.Ch != nil {
		r.Ch.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}
