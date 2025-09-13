package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/internal/core/ports"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Rabbit struct {
	Conn *amqp.Connection
	Ch   *amqp.Channel
}

var _ ports.MessageBrokerInterface = (*Rabbit)(nil)

func NewRabbitMq() (*Rabbit, error) {
	var rabbit *Rabbit
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	rabbit = &Rabbit{Conn: conn, Ch: ch}
	return rabbit, nil
}

func (r *Rabbit) SetupRabbitMQ() error {
	// --- Declare Exchanges ---
	if err := r.Ch.ExchangeDeclare(
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

	// if err := r.ch.ExchangeDeclare(
	// 	"notifications_fanout",
	// 	"fanout",
	// 	true,
	// 	false,
	// 	false,
	// 	false,
	// 	nil,
	// ); err != nil {
	// 	return err
	// }

	// // --- Declare Queues ---
	// // General kitchen queue
	// q1, err := r.ch.QueueDeclare(
	// 	"kitchen_queue", // name
	// 	true,            // durable
	// 	false,           // delete when unused
	// 	false,           // exclusive
	// 	false,           // no-wait
	// 	nil,             // arguments
	// )
	// if err != nil {
	// 	return err
	// }

	// // Bind kitchen_queue to all orders
	// if err := r.ch.QueueBind(
	// 	q1.Name,        // queue name
	// 	"kitchen.*.*",  // routing key pattern
	// 	"orders_topic", // exchange
	// 	false,
	// 	nil,
	// ); err != nil {
	// 	return err
	// }

	// Specialized queues (optional)
	// r.ch.QueueDeclare("kitchen_dine_in_queue", true, false, false, false, nil)
	// r.ch.QueueBind("kitchen_dine_in_queue", "kitchen.dine_in.*", "orders_topic", false, nil)

	// r.ch.QueueDeclare("kitchen_takeout_queue", true, false, false, false, nil)
	// r.ch.QueueBind("kitchen_takeout_queue", "kitchen.takeout.*", "orders_topic", false, nil)

	// r.ch.QueueDeclare("kitchen_delivery_queue", true, false, false, false, nil)
	// r.ch.QueueBind("kitchen_delivery_queue", "kitchen.delivery.*", "orders_topic", false, nil)

	// Notifications queue (each subscriber creates its own)
	// q2, err := r.ch.QueueDeclare(
	// 	"notifications_queue", // name
	// 	true,
	// 	false,
	// 	false,
	// 	false,
	// 	nil,
	// )
	// if err != nil {
	// 	return err
	// }

	// if err := r.ch.QueueBind(
	// 	q2.Name,
	// 	"", // fanout ignores routing key
	// 	"notifications_fanout",
	// 	false,
	// 	nil,
	// ); err != nil {
	// 	return err
	// }

	return nil
}

func (r *Rabbit) PublishOrderMessage(ctx context.Context, order domain.Order) error {
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
