package rabbitmq

import (
	"fmt"
	"log"
	"time"

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
}

func NewKitchenRabbit() (*KitchenRabbit, error) {
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

	rabbit := &KitchenRabbit{Conn: conn, Ch: ch, DurationMs: time.Duration(durationMs)}
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

func (r *KitchenRabbit) ConsumeMessages(workerName string) error {
	// GENERAL QUEUE
	_, err := r.Ch.QueueDeclare("kitchen_queue", true, false, false, false, nil)
	if err != nil {
		return err
	}
	err = r.Ch.QueueBind("kitchen_queue", "kitchen.*.*", "orders_topic", false, nil)
	if err != nil {
		return err
	}
	msgs, err := r.Ch.Consume(
		"kitchen_queue", // queue
		"",              // consumer tag
		false,           // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		return err
	}
	go r.handleMessages(msgs, workerName) // Start a goroutine for consuming messages from each queue

	// SPECIFIC QUEUES
	var specificQueues []string
	for _, orderType := range orderTypes {
		specificQueues = append(specificQueues, "kitchen_"+orderType+"_queue")
	}
	fmt.Println(specificQueues)

	for i, queueName := range specificQueues {
		_, err := r.Ch.QueueDeclare(queueName, true, false, false, false, nil)
		if err != nil {
			return err
		}
		err = r.Ch.QueueBind(specificQueues[i], "kitchen."+orderTypes[i]+".*", "orders_topic", false, nil)
		if err != nil {
			return err
		}
	}

	for _, queueName := range specificQueues {
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

		go r.handleMessages(msgs, workerName) // Start a goroutine for consuming messages from each queue
	}
	select {}
}

func (r *KitchenRabbit) handleMessages(msgs <-chan amqp.Delivery, workerName string) {
	for msg := range msgs {
		// Deserialize the message body (e.g., JSON to struct)
		orderType := string(msg.Body) // Assuming the body is a string for this example

		// Check if the worker is specialized for this order type
		if !r.isSpecializedForOrderType(orderType) {
			// Negatively acknowledge the message and requeue it
			log.Printf("Worker %s is not specialized for order type %s, rejecting message", workerName, orderType)
			msg.Nack(false, true) // requeue the message
			continue
		}

		// Process the message (your logic here)
		log.Printf("Worker %s is processing order type %s", workerName, orderType)

		// After processing, acknowledge the message
		msg.Ack(false)
	}
}

func (w *KitchenRabbit) isSpecializedForOrderType(orderType string) bool {
	for _, t := range orderTypes {
		if t == orderType {
			return true
		}
	}
	return false
}
