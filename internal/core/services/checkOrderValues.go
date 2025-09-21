package services

import (
	"fmt"
	"regexp"
	"wheres-my-pizza/internal/core/domain"
)

var validStringRegex = regexp.MustCompile(`^[a-zA-Z\s\-']{1,100}$`)

func CheckOrderValues(order domain.Order) error {
	// Customer name
	if !validStringRegex.MatchString(order.CustomerName) {
		return fmt.Errorf("invalid customer_name: must be 1–100 characters, only letters, spaces, hyphens, and apostrophes (got %s)", order.CustomerName)
	}

	// Order type
	if !(order.Type == "dine_in" || order.Type == "takeout" || order.Type == "delivery") {
		return fmt.Errorf("invalid order_type: must be one of [dine_in, takeout, delivery] (got %s)", order.Type)
	}

	// Dine-in
	if order.Type == "dine_in" && (order.TableNumber == nil || *order.TableNumber < 1 || *order.TableNumber > 100) {
		if order.TableNumber == nil {
			return fmt.Errorf("invalid table_number: number must be 1 - 100 (got nil)")
		}
		return fmt.Errorf("invalid table_number: number must be 1 - 100 (got %d)", *order.TableNumber)
	}
	if order.Type == "dine_in" && order.DeliveryAddress != nil {
		return fmt.Errorf("invalid delivery_address: must be empty when type='dine_in'")
	}

	// Delivery
	if order.Type == "delivery" && (order.DeliveryAddress == nil || len(*order.DeliveryAddress) < 10) {
		if order.DeliveryAddress == nil {
			return fmt.Errorf("invalid delivery_address: must be more than 10 characters (got nil)")
		}
		return fmt.Errorf("invalid delivery_address: must be more than 10 characters (got %d)", len(*order.DeliveryAddress))
	}
	if order.Type == "delivery" && order.TableNumber != nil {
		return fmt.Errorf("invalid table_number: must be empty when type='delivery'")
	}

	// Takeout
	if order.Type == "takeout" && order.DeliveryAddress != nil {
		return fmt.Errorf("invalid delivery_address: must be empty when type='takeout'")
	}
	if order.Type == "takeout" && order.TableNumber != nil {
		return fmt.Errorf("invalid table_number: must be empty when type='takeout'")
	}

	// Items
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
