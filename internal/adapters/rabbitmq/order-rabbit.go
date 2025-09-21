package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"wheres-my-pizza/internal/core/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type OrderRabbitInterface interface{}

var _ OrderRabbitInterface = (*OrderRabbit)(nil)

type OrderRabbit struct {
	Conn       *amqp.Connection
	Ch         *amqp.Channel
	DurationMs time.Duration
}

func NewOrderRabbit() (*OrderRabbit, error) {
	start := time.Now()
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	SetupOrderChannel(ch)

	durationMs := time.Since(start).Milliseconds()

	rabbit := &OrderRabbit{Conn: conn, Ch: ch, DurationMs: time.Duration(durationMs)}
	return rabbit, nil
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
		return fmt.Errorf("failed to publish order message: %w", err)
	}

	return nil
}

func (r *OrderRabbit) Close() {
	r.Ch.Close()
	r.Conn.Close()
}
