package entity

import "time"

// Monitor represents a URL to be monitored
type Monitor struct {
    URL      string        `json:"url"`
    Interval time.Duration `json:"interval"`
}