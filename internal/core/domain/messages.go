package domain

import "time"

type OrderDetailsResponse struct {
	OrderNumber         string     `json:"order_number"`
	CurrentStatus       string     `json:"current_status"`
	UpdatedAt           time.Time  `json:"updated_at"`
	EstimatedCompletion *time.Time `json:"estimated_completion,omitempty"`
	ProcessedBy         string     `json:"processed_by"`
}

type StatusUpdateMessage struct {
	OrderNumber         string    `json:"order_number"`
	OldStatus           string    `json:"old_status"`
	NewStatus           string    `json:"new_status"`
	ChangedBy           string    `json:"changed_by"`
	TimeStamp           time.Time `json:"timestamp"`
	EstimatedCompletion time.Time `json:"estimated_completion"`
}
