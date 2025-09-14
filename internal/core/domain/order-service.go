package domain

import "time"

type Order struct {
	ID              int       `json:"id"` // serial
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Number          string    `json:"number"` // i dont know what it is
	CustomerName    string    `json:"customer_name"`
	Type            string    `json:"order_type"` // dine_in, takeout, delivery
	TypesArray      []string
	TableNumber     *int        `json:"table_number"`     // nullable
	DeliveryAddress *string     `json:"delivery_address"` // nullable
	TotalAmount     float64     `json:"total_amount"`
	Priority        int         `json:"priority"`
	Status          string      `json:"status"`
	ProcessedBy     *string     `json:"processed_by"` // nullable
	CompletedAt     *time.Time  `json:"completed_at"` // nullable
	Items           []OrderItem `json:"items"`        // assumed sub-struct
}

type OrderItem struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	OrderID   int       `json:"order_id"`
	Name      string    `json:"name"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
}

type PutOrderResponse struct {
	OrderNumber string  `json:"order_number"`
	Status      string  `json:"status"`
	TotalAmount float64 `json:"total_amount"`
}
