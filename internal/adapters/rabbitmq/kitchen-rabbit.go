package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/pkg/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

type KitchenRabbitInterface interface{}

var _ KitchenRabbitInterface = (*KitchenRabbit)(nil)

var (
	dine_in    string   = "dine_in"
	takeout    string   = "takeout"
	delivery   string   = "delivery"
	orderTypes []string = []string{dine_in, takeout, delivery}
)

type KitchenRabbit struct {
	Conn       *amqp.Connection
	Ch         *amqp.Channel
	DurationMs time.Duration
	workerType []string
	workerName string
	logger     *logger.Logger
	qos        int
}

func NewKitchenRabbit(workerType []string, workerName string, qos int, logger *logger.Logger) (*KitchenRabbit, error) {
	rabbit := &KitchenRabbit{qos: qos, logger: logger, workerName: workerName, workerType: workerType}
	if err := rabbit.connect(); err != nil {
		return nil, err
	}

	// start reconnect watcher
	go rabbit.handleReconnect(5 * time.Second)

	return rabbit, nil
}

func (r *KitchenRabbit) connect() error {
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
	if err := setupKitchenChannel(ch, r.qos); err != nil {
		conn.Close()
		return err
	}

	r.Conn = conn
	r.Ch = ch
	r.DurationMs = time.Duration(time.Since(start).Milliseconds())

	r.logger.Info("rabbitmq", "connection_established", "Connected to RabbitMQ", map[string]interface{}{
		"worker_name": r.workerName,
	})
	return nil
}

func (r *KitchenRabbit) handleReconnect(backoff time.Duration) {
	errs := make(chan *amqp.Error)
	r.Conn.NotifyClose(errs)
	for e := range errs {
		fmt.Printf("RabbitMQ connection closed: %v. Reconnecting...\n", e)

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

func setupKitchenChannel(ch *amqp.Channel, qos int) error {
	// Orders topic
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
	err := ch.Qos(qos, 0, false) // max 10 unacknowledged messages
	if err != nil {
		return err
	}

	// Notification fanout
	if err := ch.ExchangeDeclare(
		"notifications_fanout", // name
		"fanout",               // type
		true,                   // durable
		false,                  // auto-deleted
		false,                  // internal
		false,                  // no-wait
		nil,                    // args
	); err != nil {
		return err
	}

	// Dead letter exchange
	err = ch.ExchangeDeclare(
		"orders_dlx", // DLX name
		"topic",      // exchange type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return err
	}
	return nil
}

// Dead letter queue
func (r *KitchenRabbit) dlq() (amqp.Table, error) {
	_, err := r.Ch.QueueDeclare(
		"orders_dlq", // DLQ name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return nil, err
	}

	// Bind DLQ to DLX
	err = r.Ch.QueueBind(
		"orders_dlq",
		"#",          // catch all
		"orders_dlx", // exchange
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Declare main queue with DLX policy
	args := amqp.Table{
		"x-dead-letter-exchange": "orders_dlx",
	}
	return args, nil
}

func (r *KitchenRabbit) ConsumeMessages(ctx context.Context, workerName string, errCh chan error) (chan domain.Order, error) {
	var queues []string
	for _, orderType := range orderTypes {
		queues = append(queues, "kitchen_"+orderType+"_queue")
	}
	queues = append(queues, "kitchen_queue")

	args, err := r.dlq()
	if err != nil {
		return nil, err
	}

	// Queue declaring and binding
	for i, queueName := range queues {
		_, err := r.Ch.QueueDeclare(queueName, true, false, false, false, args)
		if err != nil {
			return nil, err
		}
		if queueName == "kitchen_queue" {
			err = r.Ch.QueueBind(queues[i], "kitchen.*", "orders_topic", false, nil)
		} else {
			err = r.Ch.QueueBind(queues[i], "kitchen."+orderTypes[i]+".*", "orders_topic", false, nil)
		}
		if err != nil {
			return nil, err
		}
	}

	orderCh := make(chan domain.Order)
	// Consuming messages
	for _, queueName := range queues {
		msgs, err := r.Ch.Consume(
			queueName, // queue
			"",        // consumer tag
			false,     // auto-ack
			false,     // exclusive
			false,     // no-local
			false,     // no-wait
			nil,       // args
		)
		if err != nil {
			log.Fatalf("Failed to register a consumer for queue %s: %s", queueName, err)
		}

		go r.handleMessages(msgs, orderCh, errCh) // Start a goroutine for consuming messages from each queue
	}

	return orderCh, nil
}

func (r *KitchenRabbit) PublishStatusUpdateMessage(ctx context.Context, order domain.Order, oldOrderStatus, workerName string, seconds int) error {
	t1 := time.Now()
	t2 := t1.Add(time.Duration(seconds) * time.Second)
	msg := domain.StatusUpdateMessage{OrderNumber: order.Number, OldStatus: oldOrderStatus, NewStatus: order.Status, ChangedBy: workerName, TimeStamp: t1, EstimatedCompletion: t2}
	body, err := json.Marshal(msg)
	fmt.Println(string(body))
	if err != nil {
		return fmt.Errorf("failed to marshal order message: %w", err)
	}

	// Publish to exchange
	err = r.Ch.PublishWithContext(
		ctx,                    // context
		"notifications_fanout", // exchange
		"",                     // routing key
		false,                  // mandatory
		false,                  // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // make message persistent
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish order message: %w", err)
	}

	return nil
}

func (r *KitchenRabbit) handleMessages(msgs <-chan amqp.Delivery, orderCh chan<- domain.Order, errCh <-chan error) error {
	for msg := range msgs {

		order := domain.Order{}
		err := json.Unmarshal(msg.Body, &order)
		if err != nil {
			return err
		}
		// Check if the worker is specialized for this order type
		if !r.isSpecializedForOrderType(order.Type) {
			msg.Nack(false, true) // requeue the message
			continue
		}

		r.logger.Debug(order.Number, "order_processing_started", "Order is picked from the queue", map[string]interface{}{"worker_name": r.workerName})

		// After processing, acknowledge the message
		orderCh <- order
		err = <-errCh
		if err != nil {
			r.logger.Error(order.Number, "message_processing_failed", "Unrecoverable processing errors", err, map[string]interface{}{"worker_name": r.workerName})
			msg.Nack(false, true)
		} else {
			r.logger.Debug(order.Number, "order_completed", "Order is fully processed", map[string]interface{}{"worker_name": r.workerName})
			msg.Ack(false)
		}

	}
	return nil
}

func (r *KitchenRabbit) isSpecializedForOrderType(orderType string) bool {
	for _, t := range r.workerType {
		if t == orderType {
			return true
		}
	}
	return false
}

func (r *KitchenRabbit) Close() {
	r.Ch.Close()
	r.Conn.Close()
}
