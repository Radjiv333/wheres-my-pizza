package services

import "wheres-my-pizza/internal/core/domain"

func AssignPriority(order domain.Order) int {
	var totalAmount float64
	for _, item := range order.Items {
		totalAmount += float64(item.Quantity) * item.Price
	}

	switch {
	case totalAmount > 100:
		return 10
	case totalAmount >= 50:
		return 5
	default:
		return 1
	}
}
