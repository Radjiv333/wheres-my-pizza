package rabbitmq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"wheres-my-pizza/internal/core/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type KitchenRabbitInterface interface{}

var _ KitchenRabbitInterface = (*KitchenRabbit)(nil)

var (
	dine_in            string   = "dine_in"
	takeout            string   = "takeout"
	delivery           string   = "delivery"
	orderTypes         []string = []string{dine_in, takeout, delivery}
	numberOfOrderTypes int      = 3
)

type KitchenRabbit struct {
	Conn       *amqp.Connection
	Ch         *amqp.Channel
	DurationMs time.Duration
	workerType []string
	workerName string
}

func NewKitchenRabbit(workerType []string, workerName string) (*KitchenRabbit, error) {
	start := time.Now()
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	SetupKitchenChannel(ch)

	durationMs := time.Since(start).Milliseconds()

	rabbit := &KitchenRabbit{Conn: conn, Ch: ch, DurationMs: time.Duration(durationMs), workerType: workerType, workerName: workerName}
	return rabbit, nil
}

func SetupKitchenChannel(ch *amqp.Channel) error {
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
	err := ch.Qos(10, 0, false) // max 10 unacknowledged messages
	if err != nil {
		return err
	}

	return nil
}

func (r *KitchenRabbit) ConsumeMessages(ctx context.Context, workerName string) (chan domain.Order, error) {
	var queues []string
	for _, orderType := range orderTypes {
		queues = append(queues, "kitchen_"+orderType+"_queue")
	}
	queues = append(queues, "kitchen_queue")

	// Queue declaring and binding
	for i, queueName := range queues {
		_, err := r.Ch.QueueDeclare(queueName, true, false, false, false, nil)
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
	ch := make(chan domain.Order)
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

		go r.handleMessages(msgs, ch) // Start a goroutine for consuming messages from each queue
	}

	return ch, nil
}

func (r *KitchenRabbit) handleMessages(msgs <-chan amqp.Delivery, ch chan<- domain.Order) error {
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

		// Process the message (your logic here)
		log.Printf("Worker %s is processing order type %s", r.workerName, order.Type)

		// After processing, acknowledge the message
		msg.Ack(false)
		ch <- order
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
