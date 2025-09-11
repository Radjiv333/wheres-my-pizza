package services

import (
	"flag"
	"fmt"
	"os"
)

type Flags struct {
	Port          int
	MaxConcurrent int
}

func FlagParse() (Flags, error) {
	help := flag.Bool("help", false, "Shows usage to the screen")

	// Order-service, Tracking-service, Notification-service
	mode := flag.String("mode", "", "Establishing the working mode for the app.")
	port := flag.Int("port", 0, "The HTTP port for the API.")
	maxConcurrent := flag.Int("max-concurrent", 50, "Maximum number of concurrent orders to process.")

	// Kitchen-service
	workerName := flag.String("worker-name", "", "Unique name for worker")
	orderTypes := flag.String("order-types", "", "Optional. Comma-separated list of order types the worker can handle (e.g., dine_in,takeout). If omitted, handles all.")
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

	err := CheckFlags(*mode, *workerName, *orderTypes, *port, *maxConcurrent, *heartbeatInterval, *prefetch, isSetByUser)
	if err != nil {
		return Flags{}, nil
	}

	switch *mode {
	case "order-service":
		if !isSetByUser {
			*port = 3000
		}
		return Flags{Port: *port, MaxConcurrent: *maxConcurrent}, nil
	case "kitchen-worker":

	case "tracking-service":

	case "notification-subscriber":
	default:
		// ERROR LOGGER
		fmt.Println("Something is wrong with mode")
		os.Exit(1)

	}
	return Flags{}, nil
}
