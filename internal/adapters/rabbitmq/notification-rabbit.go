package rabbitmq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/pkg/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationRabbit struct {
	Conn       *amqp.Connection
	Ch         *amqp.Channel
	DurationMs time.Duration
	logger     *logger.Logger
}

func NewNotificationRabbit(logger *logger.Logger) (*NotificationRabbit, error) {
	start := time.Now()
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	err = setupNotificationChannel(ch)
	if err != nil {
		return nil, err
	}

	durationMs := time.Since(start).Milliseconds()

	rabbit := &NotificationRabbit{Conn: conn, Ch: ch, DurationMs: time.Duration(durationMs), logger: logger}
	return rabbit, nil
}

func setupNotificationChannel(ch *amqp.Channel) error {
	// Orders topic
	err := ch.ExchangeDeclare(
		"notifications_fanout", // exchange name
		"fanout",               // type
		true,                   // durable
		false,                  // auto-deleted
		false,                  // internal
		false,                  // no-wait
		nil,                    // args
	)
	return err
}

func (r *NotificationRabbit) ConsumeMessages(ctx context.Context) error {
	// Declare queue
	q, err := r.Ch.QueueDeclare(
		"notifications_queue", // queue name
		true,                  // durable
		false,                 // delete when unused
		false,                 // exclusive
		false,                 // no-wait
		nil,                   // args
	)
	if err != nil {
		return err
	}

	// Bind queue to exchange
	err = r.Ch.QueueBind(
		q.Name,                 // queue name
		"",                     // routing key (ignored for fanout)
		"notifications_fanout", // exchange
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Consume messages
	msgs, err := r.Ch.Consume(
		q.Name,
		"",
		false, // auto-ack = false, we will ack manually
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Println("📡 Notification Service is running...")

	for {
		select {
		case d := <-msgs:
			var msg domain.OrderDetailsResponse

			err := json.Unmarshal(d.Body, &msg)
			if err != nil {
				log.Printf("failed to unmarshal message: %v", err)
				d.Nack(false, false) // reject, don’t requeue
				continue
			}
			// Acknowledge message
			d.Ack(false)
		case <-ctx.Done():
			log.Println("Notification Service shutting down...")
			return nil
		}
	}
}
