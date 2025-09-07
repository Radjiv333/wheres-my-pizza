package services

import "fmt"

var appUsage string = `Usage:
  ./restaurant-system [--mode <S>] [options]
  ./restaurant-system --help

Options:
  --help       Show this screen.
  --mode S                Required. Restaurant mode. Possible mode options (S): "order-service", "kitchen-worker", "tracking-service", "notification-subscriber".

'Order-service' service Options:
  --port N                Default: 3000. Port number. Port number 'N' must be between 1024 and 49151 inclusively.
  
'Kitchen-worker' service Options:
  --worker-name S         Required. Establishes unique name for the worker.
  --order-types S         Optional. Comma-separated list of order types the worker can handle (e.g., dine_in,takeout). If omitted, handles all.
  --heartbeat-interval N  Default: 30s. Interval (seconds) between heartbeats.
  --prefetch N            Default: 1. RabbitMQ prefetch count, limiting how many messages the worker receives at once.  
  
'Tracking-service' service Options:
  --port N                Default: 3000. Port number. Port number 'N' must be between 1024 and 49151 inclusively.
`

func AppUsage() {
	fmt.Print(appUsage)
}
