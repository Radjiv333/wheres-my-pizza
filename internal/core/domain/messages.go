package domain

import "time"

type OrderDetailsResponse struct {
	OrderNumber         string     `json:"order_number"`
	CurrentStatus       string     `json:"current_status"`
	UpdatedAt           time.Time  `json:"updated_at"`
	EstimatedCompletion *time.Time `json:"estimated_completion,omitempty"`
	ProcessedBy         string     `json:"processed_by"`
}

type LogEntry struct {
	Timestamp string                 `json:"timestamp"` // ISO 8601 format
	Level     string                 `json:"level"`     // INFO, DEBUG, ERROR
	Service   string                 `json:"service"`   // e.g. notification-subscriber
	Hostname  string                 `json:"hostname"`  // container or host
	RequestID string                 `json:"request_id"`
	Action    string                 `json:"action"`  // short machine-readable action
	Message   string                 `json:"message"` // human-readable description
	Details   map[string]interface{} `json:"details,omitempty"`
}
