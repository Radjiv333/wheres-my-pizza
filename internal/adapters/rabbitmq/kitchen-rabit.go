package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
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

func (r *KitchenRabbit) ConsumeMessages(ctx context.Context, workerName string, errCh chan error) (chan domain.Order, error) {
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

		// Process the message
		log.Printf("Worker %s is processing order type %s", r.workerName, order.Type)

		// After processing, acknowledge the message
		orderCh <- order
		fmt.Println("waiting")
		err = <-errCh
		fmt.Println("waited for err")
		if err != nil {
			fmt.Printf("error encountered: %v\n", err)
			msg.Nack(false, true)
		} else {
			fmt.Println("im in ack")
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

type StatusUpdateMessage struct {
	OrderNumber         string    `json:"order_number"`
	OldStatus           string    `json:"old_status"`
	NewStatus           string    `json:"new_status"`
	ChangedBy           string    `json:"changed_by"`
	TimeStamp           time.Time `json:"timestamp"`
	EstimatedCompletion time.Time `json:"estimated_completion"`
}

func (r *KitchenRabbit) PublishStatusUpdateMessage(ctx context.Context, order domain.Order, newOrderStatus, workerName string, seconds int) error {
	t1 := time.Now()
	t2 := t1.Add(time.Duration(seconds) * time.Second)
	msg := StatusUpdateMessage{OrderNumber: order.Number, OldStatus: order.Status, NewStatus: newOrderStatus, ChangedBy: workerName, TimeStamp: t1, EstimatedCompletion: t2}
	body, err := json.Marshal(msg)
	fmt.Println(string(body))
	if err != nil {
		return fmt.Errorf("failed to marshal order message: %w", err)
	}
	// -------------------------------------------------------------------------------------------------------------------------------------------------------
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
