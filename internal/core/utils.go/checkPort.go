package utils

import (
	"errors"
	"fmt"
)

func CheckPort(port int, isSetByUser bool) error {
	if (port > 49151 || port < 1024) && isSetByUser {
		// ERROR LOGGER
		errMessage := fmt.Sprintf("invalid 'port' value: %d", port)
		return errors.New(errMessage)
	}
	return nil
}
