# wheres-my-pizza - Distributed Restaurant Order Management System

**wheres-my-pizza** is a distributed restaurant order management system built in Go. It simulates a real-world restaurant workflow, from order placement via an API, to kitchen processing, to order tracking and notifications. The system leverages microservices, **RabbitMQ** for asynchronous messaging, and **PostgreSQL** for persistent storage.

This project teaches key modern software engineering concepts: **microservices architecture, message queue patterns, concurrent programming, and scalable system design**.

---

## Features

- Microservices Architecture
- Message Queue Systems
- RabbitMQ Integration
- Concurrent Programming
- Transactional Database Operations
- Graceful Shutdown and Logging
- Logging

---

## Technologies Used

- **Language:** Go
- **Database:** PostgreSQL
- **Message Broker:** RabbitMQ
- **Containerization:** Docker, Docker Compose
- **Concurrency:** Goroutines, Worker Pools
- **Code Formatting:** gofumpt
- **Logging:** Structured JSON logs
- **AMQP Client:** `github.com/rabbitmq/amqp091-go`
- **PostgreSQL Driver:** `pgx/v5`

---

## System Architecture Overview

Application consists of four main services, a database, and a message broker. The services communicate asynchronously via RabbitMQ:

```
                                +--------------------------------------------+
                                |               PostgreSQL DB                |
                                |             (Order Storage)                |
                                +--+-------------+---------------------------+
                                   ^             ^                    |
                  (Writes & Reads) |             | (Writes & Reads)   |
                                   v             v                    |
+------------+        +-----------+              +---------------+    |
| HTTP Client|------->|  Order    |              | Kitchen       |    |
| (e.g. curl)|        |  Service  |              | Service       |    |
+------------+        +---------- +              +-+-------------+    |
                         |                         ^                  |
                (Publishes New Order)    (Publishes Status Update)    |
                         v                         |                  |
                   +-----+-------------------------+---------+        |
                   |                                         |        |
                   |         RabbitMQ Message Broker         |        |
                   |                                         |        |
                   +-----------------------------------------+        |
                              |                                       |
                              | (Status Updates)                      | (Reads)
                              v                                       v
                        +-----+-----------+         +-----+------------------+
                        | Notification    |         | Tracking               |
                        | Subscriber      |         | Service                |
                        +-----------------+         +------------------------+
```

---

## Services

- **Order Service**: Receives new orders, validates input, persists to PostgreSQL, and publishes messages to RabbitMQ.
- **Kitchen Worker**: Consumes order messages, processes cooking logic, updates statuses, and publishes status updates.
- **Tracking Service**: Read-only API to track orders and view kitchen worker status.
- **Notification Subscriber**: Subscribes to updates and prints notifications, demonstrating fanout messaging.

---

## Prerequisites

- Go 1.23+
- Docker
- Docker Compose
- Running PostgreSQL instance
- Running RabbitMQ instance

---

## Getting Started

### Using Makefile

```bash
# Build the project
make build

# Start services in attached mode
make up
````

### Running Individual Services

```bash
# Order Service
./restaurant-system --mode=order-service --port=3000

# Kitchen Worker
./restaurant-system --mode=kitchen-worker --worker-name="chef_anna" --prefetch=1
./restaurant-system --mode=kitchen-worker --worker-name="chef_mario" --order-types="dine_in" &

# Tracking Service
./restaurant-system --mode=tracking-service --port=3002

# Notification Subscriber
./restaurant-system --mode=notification-subscriber
```

---

## API Usage

### Order Service Endpoints

**POST /orders**

```json
{
  "customer_name": "John Doe",
  "order_type": "takeout",
  "items": [
    { "name": "Margherita Pizza", "quantity": 1, "price": 15.99 },
    { "name": "Caesar Salad", "quantity": 1, "price": 8.99 }
  ]
}
```

### Tracking Service Endpoints

* **GET /orders/{order_number}/status**: Retrieve current order status.
* **GET /orders/{order_number}/history**: Retrieve full order history.
* **GET /workers/status**: Retrieve all kitchen workers’ status.

---

## Configuration

Use `config.yaml` to set up database and RabbitMQ connection details:

```yaml
database:
  host: localhost
  port: 5432
  user: restaurant_user
  password: restaurant_pass
  database: restaurant_db

rabbitmq:
  host: localhost
  port: 5672
  user: guest
  password: guest
```

---

## Important Notes

* RabbitMQ connections handle **reconnection scenarios**.
* All database operations are **transactional**.
* Structured JSON logging is **consistent for all services**.
