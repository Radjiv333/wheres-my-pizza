Absolutely! Based on your Makefile and manual, I’ve drafted a comprehensive, professional README for **wheres-my-pizza**. I’ve structured it for clarity, added a **Technologies Used** section, and highlighted the **microservices architecture and message queue patterns**.

---

```markdown
# wheres-my-pizza - Distributed Restaurant Order Management System

**wheres-my-pizza** is a distributed restaurant order management system built in Go. It simulates a real-world restaurant workflow, from order placement via an API, to kitchen processing, to order tracking and notifications. The system leverages microservices, RabbitMQ for asynchronous messaging, and PostgreSQL for persistent storage.

This project teaches key modern software engineering concepts: **microservices architecture, message queue patterns, concurrent programming, and scalable system design**.

---

## Learning Objectives

- Microservices Architecture
- Message Queue Systems
- RabbitMQ Integration
- Concurrent Programming
- Transactional Database Operations
- Graceful Shutdown and Logging

---

## Technologies Used

- **Language:** Go
- **Database:** PostgreSQL
- **Message Broker:** RabbitMQ
- **Containerization:** Docker, Docker Compose
- **Architecture:** Microservices
- **Concurrency:** Goroutines, Worker Pools
- **Code Formatting:** gofumpt
- **Logging:** Structured JSON logs
- **AMQP Client:** `github.com/rabbitmq/amqp091-go`
- **PostgreSQL Driver:** `pgx/v5`

---

## System Architecture Overview

Your application consists of four main services, a database, and a message broker. The services communicate asynchronously via RabbitMQ:

```

```
                            +--------------------------------------------+
                            |               PostgreSQL DB                |
                            |             (Order Storage)                |
                            +--+-------------+---------------------------+
                               ^             ^                    |
              (Writes & Reads) |             | (Writes & Reads)   |
                               v             v                    |
```

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

````

---

## Features

- **Order Service**: Receives new orders, validates input, persists to PostgreSQL, and publishes messages to RabbitMQ.
- **Kitchen Worker**: Consumes order messages, processes cooking logic, updates statuses, and publishes status updates.
- **Tracking Service**: Read-only API to track orders and view kitchen worker status.
- **Notification Subscriber**: Subscribes to updates and prints notifications, demonstrating fanout messaging.
- **Message Queue Patterns**:
  - **Work Queue:** Distribute tasks among workers for load balancing.
  - **Publish/Subscribe:** Broadcast order status updates.
  - **Routing:** Route messages to specialized queues/workers based on order type and priority.

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

# Start services in detached mode
make upd

# Stop services
make down

# Restart services
make restart

# Remove all containers, networks, and volumes
make nuke
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

### Create a New Order

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

**Response:**

```json
{
  "order_number": "ORD_20241216_001",
  "status": "received",
  "total_amount": 24.98
}
```

---

### Tracking Service Endpoints

* **GET /orders/{order_number}/status**: Retrieve current order status.
* **GET /orders/{order_number}/history**: Retrieve full order history.
* **GET /workers/status**: Retrieve all kitchen workers’ status.

---

## Database Schema Highlights

**Orders Table**

```sql
create table "orders" (
  "id" serial primary key,
  "created_at" timestamptz default now(),
  "updated_at" timestamptz default now(),
  "number" text unique not null,
  "customer_name" text not null,
  "type" text not null check (type in ('dine_in','takeout','delivery')),
  "table_number" integer,
  "delivery_address" text,
  "total_amount" decimal(10,2) not null,
  "priority" integer default 1,
  "status" text default 'received',
  "processed_by" text,
  "completed_at" timestamptz
);
```

**Order Items Table**

```sql
create table order_items (
  "id" serial primary key,
  "created_at" timestamptz default now(),
  "order_id" integer references orders(id),
  "name" text not null,
  "quantity" integer not null,
  "price" decimal(8,2) not null
);
```

**Order Status Log Table**

```sql
create table order_status_log (
  "id" serial primary key,
  "created_at" timestamptz default now(),
  "order_id" integer references orders(id),
  "status" text,
  "changed_by" text,
  "changed_at" timestamptz default current_timestamp,
  "notes" text
);
```

---

## Logging

* Structured JSON logs
* Mandatory fields: `timestamp, level, service, action, message, hostname, request_id`
* Error object includes `msg` and `stack`

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

* All services must implement **graceful shutdown**.
* RabbitMQ connections must handle **reconnection scenarios**.
* All database operations must be **transactional**.
* Structured JSON logging is mandatory for all services.

---

## References

* [RabbitMQ Documentation](https://www.rabbitmq.com/documentation.html)
* [Go AMQP Client](https://pkg.go.dev/github.com/rabbitmq/amqp091-go)
* [PostgreSQL pgx/v5 Driver](https://github.com/jackc/pgx)
* [Gofumpt Formatter](https://github.com/mvdan/gofumpt)

```

---

This README now includes:  

✅ Project overview and purpose  
✅ Learning objectives  
✅ Technologies used  
✅ Architecture diagram and explanation  
✅ Detailed API and usage instructions  
✅ Database schema  
✅ Logging standards  
✅ Configuration  
✅ Makefile usage  

---

If you want, I can also **create a short “Quick Start Diagram” showing how messages flow through RabbitMQ between services**, which would make the README even more visually intuitive.  

Do you want me to do that?
```
