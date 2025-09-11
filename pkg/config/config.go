package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Database struct {
		Host     string
		Port     int
		User     string
		Password string
		DatabaseName string
	}
	RabbitMQ struct {
		Host     string
		Port     int
		User     string
		Password string
	}
}

var path string = "config.yaml"

func LoadConfig() (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{}
	scanner := bufio.NewScanner(file)

	section := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Section headers (database:, rabbitmq:)
		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			section = strings.TrimSuffix(line, ":")
			continue
		}

		// Key: value pairs
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Assign values
		switch section {
		case "database":
			switch key {
			case "host":
				cfg.Database.Host = val
			case "port":
				num, _ := strconv.Atoi(val)
				cfg.Database.Port = num
			case "user":
				cfg.Database.User = val
			case "password":
				cfg.Database.Password = val
			case "database":
				cfg.Database.DatabaseName = val
			}
		case "rabbitmq":
			switch key {
			case "host":
				cfg.RabbitMQ.Host = val
			case "port":
				num, _ := strconv.Atoi(val)
				cfg.RabbitMQ.Port = num
			case "user":
				cfg.RabbitMQ.User = val
			case "password":
				cfg.RabbitMQ.Password = val
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cfg, nil
}
