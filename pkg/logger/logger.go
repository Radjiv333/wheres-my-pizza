package logger

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"time"
)

// LogEntry defines the structure of a log
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Hostname  string                 `json:"hostname"`
	RequestID string                 `json:"request_id,omitempty"`
	Action    string                 `json:"action"`
	Message   string                 `json:"message"`
	Error     *ErrorObject           `json:"error,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// ErrorObject for structured error logging
type ErrorObject struct {
	Msg   string `json:"msg"`
	Stack string `json:"stack"`
}

// Logger holds service name and hostname
type Logger struct {
	service  string
	hostname string
}

// New initializes a new Logger
func NewLogger(service string) *Logger {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = getFallbackHostname()
	}
	return &Logger{
		service:  service,
		hostname: hostname,
	}
}

// Info logs an INFO message
func (l *Logger) Info(requestID, action, message string, extra map[string]interface{}) {
	l.log("INFO", action, message, requestID, nil, extra)
}

// Debug logs a DEBUG message
func (l *Logger) Debug(requestID, action, message string, extra map[string]interface{}) {
	l.log("DEBUG", action, message, requestID, nil, extra)
}

// Error logs an ERROR message with error object
func (l *Logger) Error(requestID, action, message string, err error, extra map[string]interface{}) {
	errorObj := &ErrorObject{
		Msg:   err.Error(),
		Stack: string(debug.Stack()),
	}
	l.log("ERROR", action, message, requestID, errorObj, extra)
}

// Internal function to build and print JSON logs
func (l *Logger) log(level, action, message, requestID string, errObj *ErrorObject, extra map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Service:   l.service,
		Hostname:  l.hostname,
		RequestID: requestID,
		Action:    action,
		Message:   message,
		Error:     errObj,
		Extra:     extra,
	}

	b, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal log entry: %v\n", err)
		return
	}
	fmt.Println(string(b))
}

// Fallback if os.Hostname() fails
func getFallbackHostname() string {
	addrs, _ := net.InterfaceAddrs()
	if len(addrs) > 0 {
		return addrs[0].String()
	}
	return "unknown-host"
}
