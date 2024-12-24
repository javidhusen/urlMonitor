package entity

import "time"

// LogEntry represents a single monitoring log entry
type LogEntry struct {
    Timestamp    time.Time `json:"timestamp"`
    URL          string    `json:"url"`
    StatusCode   int       `json:"statusCode"`
    ResponseTime int64     `json:"responseTime"` // in milliseconds
    Success      bool      `json:"success"`
    Error        string    `json:"error,omitempty"`
}