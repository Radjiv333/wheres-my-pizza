package services

import (
	"flag"
	"os"
)

func FlagParse() error {
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

	if *help {
		AppUsage()
		os.Exit(0)
	}

	err := CheckFlags(*mode, *workerName, *orderTypes, *port, *maxConcurrent, *heartbeatInterval, *prefetch)
	if err != nil {
		return err
	}

	return nil
}
