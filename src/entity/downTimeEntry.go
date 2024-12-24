package entity

import "time"

// DowntimeEntry represents a period of downtime for a URL
type DowntimeEntry struct {
    URL         string    `json:"url"`
    StartTime   time.Time `json:"startTime"`
    EndTime     time.Time `json:"endTime"`
    Duration    string    `json:"duration"`
    StatusCode  int       `json:"statusCode"`
    ErrorDetail string    `json:"errorDetail,omitempty"`
}