package services

import (
	"flag"
	"fmt"
	"os"

	"wheres-my-pizza/internal/core/utils.go"
)

type KitchenFlags struct {
	WorkerName        string
	OrderTypes        []string
	HeartbeatInterval int
	Prefetch          int
}
type OrderFlags struct {
	Port          int
	MaxConcurrent int
}
type Flags struct {
	Mode    string
	Order   OrderFlags
	Kitchen KitchenFlags
}

func FlagParse() (Flags, error) {
	help := flag.Bool("help", false, "Shows usage to the screen")

	// Order-service, Tracking-service, Notification-service
	mode := flag.String("mode", "", "Establishing the working mode for the app.")
	port := flag.Int("port", 0, "The HTTP port for the API.")
	maxConcurrent := flag.Int("max-concurrent", 50, "Maximum number of concurrent orders to process.")

	// Kitchen-service
	workerName := flag.String("worker-name", "", "Unique name for worker")
	orderTypes := flag.String("order-types", "takeout, dine_in, delivery", "Optional. Comma-separated list of order types the worker can handle (e.g., dine_in,takeout). If omitted, handles all.")
	heartbeatInterval := flag.Int("heartbeat-interval", 30, "Maximum number of concurrent orders to process.")
	prefetch := flag.Int("prefetch", 1, "RabbitMQ prefetch count, limiting how many messages the worker receives at once.")

	flag.Parse()

	isSetByUser := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			isSetByUser = true
		}
	})

	if *help {
		AppUsage()
		os.Exit(0)
	}

	// Checking for flag values
	err := CheckFlags(*mode, *workerName, *orderTypes, *port, *maxConcurrent, *heartbeatInterval, *prefetch, isSetByUser)
	if err != nil {
		return Flags{}, err
	}

	// Return 'Flags' struct
	switch *mode {
	case "order-service":
		if !isSetByUser {
			*port = 3000
		}
		orderFlags := OrderFlags{Port: *port, MaxConcurrent: *maxConcurrent}
		return Flags{Mode: *mode, Order: orderFlags}, nil
	case "kitchen-worker":
		orderTypesArr := utils.GetStringArray(*orderTypes)
		kitchenFlags := KitchenFlags{WorkerName: *workerName, OrderTypes: orderTypesArr, HeartbeatInterval: *heartbeatInterval, Prefetch: *prefetch}
		return Flags{Mode: *mode, Kitchen: kitchenFlags}, nil
	case "tracking-service":
		if !isSetByUser {
			*port = 3002
		}
		orderFlags := OrderFlags{Port: *port}
		return Flags{Mode: *mode, Order: orderFlags}, nil
	case "notification-subscriber":
	default:
		// ERROR LOGGER
		fmt.Println("Something is wrong with mode")
		os.Exit(1)

	}
	return Flags{}, nil
}
