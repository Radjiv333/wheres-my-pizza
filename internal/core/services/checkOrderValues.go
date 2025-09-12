package services

import (
	"fmt"
	"regexp"

	"wheres-my-pizza/internal/core/domain"
)

var validStringRegex = regexp.MustCompile(`^[a-zA-Z\s\-']{1,100}$`)

func CheckOrderValues(order domain.Order) error {
	if !validStringRegex.MatchString(order.CustomerName) {
		return fmt.Errorf("invalid customer_name: must be 1–100 characters, only letters, spaces, hyphens, and apostrophes (got %s)", order.CustomerName)
	}
	if !(order.Type == "dine_in" || order.Type == "takeout" || order.Type == "delivery") {
		return fmt.Errorf("invalid order_type: must be one of [dine_in, takeout, delivery] (got %s)", order.Type)
	}
	if len(order.Items) < 1 || len(order.Items) > 20 {
		return fmt.Errorf("items count is invalid: got %d, allowed 1 - 20", len(order.Items))
	}

	// Validate each item
	for i, item := range order.Items {
		if len(item.Name) < 1 || len(item.Name) > 50 {
			return fmt.Errorf("item[%d].name is invalid: length must be 1 - 50", i)
		}
		if item.Quantity < 1 || item.Quantity > 10 {
			return fmt.Errorf("item[%d].quantity is invalid: got %d, allowed 1 - 10", i, item.Quantity)
		}
		if item.Price < 0.01 || item.Price > 999.99 {
			return fmt.Errorf("item[%d].price is invalid: got %.2f, allowed 0.01 - 999.99", i, item.Price)
		}
	}

	return nil
}
