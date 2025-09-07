package orders

import (
	"fmt"

	"wheres-my-pizza/internal/core/ports"
)

type Orders struct {
	maxConcurrent int
}

var _ ports.OrderService = (*Orders)(nil)

func (o *Orders) Stop() {
	fmt.Println("Stop function")
}
