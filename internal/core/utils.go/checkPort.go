package utils

import (
	"errors"
	"fmt"
)

func CheckPort(port int) error {
	if port > 49151 || port < 1024 {
		// ERROR LOGGER
		errMessage := fmt.Sprintf("invalid 'port' value: %d", port)
		return errors.New(errMessage)
	}
	return nil
}
