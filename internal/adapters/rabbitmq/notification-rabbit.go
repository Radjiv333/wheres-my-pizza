package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationRabbit struct {
	Conn       *amqp.Connection
	Ch         *amqp.Channel
	DurationMs time.Duration
	logger     *logger.Logger
	url string
}

func NewNotificationRabbit(logger *logger.Logger, cfg config.Config) (*NotificationRabbit, error) {
	rabbitURL := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		cfg.RabbitMQ.User, cfg.RabbitMQ.Password, cfg.RabbitMQ.Host,
		cfg.RabbitMQ.Port)
	rabbit := &NotificationRabbit{logger: logger, url: rabbitURL}
	if err := rabbit.connect(); err != nil {
		return nil, err
	}

	// start reconnect watcher
	go rabbit.handleReconnect(5 * time.Second)

	return rabbit, nil
}

func (r *NotificationRabbit) connect() error {
	start := time.Now()

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}
	if err := setupNotificationChannel(ch); err != nil {
		conn.Close()
		return err
	}

	r.Conn = conn
	r.Ch = ch
	r.DurationMs = time.Duration(time.Since(start).Milliseconds())

	r.logger.Info("rabbitmq", "connection_established", "Connected to RabbitMQ (notification)", nil)
	return nil
}

func (r *NotificationRabbit) handleReconnect(backoff time.Duration) {
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

	for {
		select {
		case d := <-msgs:
			var msg domain.StatusUpdateMessage

			err := json.Unmarshal(d.Body, &msg)
			if err != nil {
				log.Printf("failed to unmarshal message: %v", err)
				d.Nack(false, false) // reject, don’t requeue
				continue
			}

			// Acknowledge message
			r.logger.Info("", "notification_received", "Status update message is received", map[string]interface{}{"details": map[string]interface{}{"order_number": msg.OrderNumber, "new_status": msg.NewStatus}})
			fmt.Printf("Notification for order %s: Status changed from '%s' to '%s' by %s.\n", msg.OrderNumber, msg.OldStatus, msg.NewStatus, msg.ChangedBy)
			d.Ack(false)
		case <-ctx.Done():
			log.Println("Notification Service shutting down...")
			return nil
		}
	}
}
