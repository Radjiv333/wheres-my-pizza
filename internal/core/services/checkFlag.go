package services

import (
	"errors"
	"fmt"

	"wheres-my-pizza/internal/core/utils.go"
)

func CheckFlags(mode, workerName, orderTypes string, port, maxConcurrent, heartbeatInterval, prefetch int, isSetByUser bool) error {
	switch mode {
	case "order-service":
		if err := utils.CheckPort(port, isSetByUser); err != nil {
			return err
		}
		if maxConcurrent <= 0 || maxConcurrent > 100 {
			errMessage := fmt.Sprintf("invalid 'max-concurrent' value: %d", maxConcurrent)
			return errors.New(errMessage)
		}
	case "kitchen-worker":
		if workerName == "" {
			errMessage := "'worker-name' value cannot be empty"
			return errors.New(errMessage)
		}

		orderTypesArr := utils.GetStringArray(orderTypes)
		if len(orderTypesArr) == 0 {
			errMessage := "invalid 'order-types' value: value is empty"
			return errors.New(errMessage)
		}
		for _, orderType := range orderTypesArr {
			if !(orderType == "dine_in" || orderType == "delivery" || orderType == "takeout") {
				errMessage := fmt.Sprintf("invalid 'order-types' value: %s", orderType)
				return errors.New(errMessage)
			}
		}
		if heartbeatInterval <= 0 || heartbeatInterval > 50 {
			errMessage := fmt.Sprintf("invalid 'heartbeat-interval' value: %d", heartbeatInterval)
			return errors.New(errMessage)
		}
		if prefetch <= 0 || prefetch > 10 {
			errMessage := fmt.Sprintf("invalid 'prefetch' value: %d", prefetch)
			return errors.New(errMessage)
		}
	case "tracking-service":
		if err := utils.CheckPort(port, isSetByUser); err != nil {
			return err
		}
	case "notification-subscriber":
	default:
		errMessage := fmt.Sprintf("invalid mode' value: %s", mode)
		return errors.New(errMessage)
	}
	return nil
}
