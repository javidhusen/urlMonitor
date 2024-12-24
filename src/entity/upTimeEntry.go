package entity

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type UptimeMonitor struct {
	monitors     map[string]Monitor
	logs         []LogEntry
	downtimes    []DowntimeEntry
	stopChannels map[string]chan struct{}
	mu           sync.RWMutex
	client       *http.Client
}

func NewUptimeMonitor() *UptimeMonitor {
    return &UptimeMonitor{
        monitors:     make(map[string]Monitor),
        logs:         make([]LogEntry, 0),
        downtimes:    make([]DowntimeEntry, 0),
        stopChannels: make(map[string]chan struct{}),
        client: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (um *UptimeMonitor) AddMonitor(url string, interval time.Duration) error {
    um.mu.Lock()
    defer um.mu.Unlock()

    if interval == 0 {
        interval = 30 * time.Second
    }

    if _, exists := um.monitors[url]; exists {
        return fmt.Errorf("URL %s is already being monitored", url)
    }

    um.monitors[url] = Monitor{URL: url, Interval: interval}
    stopChan := make(chan struct{})
    um.stopChannels[url] = stopChan

    go um.monitorURL(url, interval, stopChan)
    return nil
}

func (um *UptimeMonitor) RemoveMonitor(url string) error {
    um.mu.Lock()
    defer um.mu.Unlock()

    if stopChan, exists := um.stopChannels[url]; exists {
        close(stopChan)
        delete(um.stopChannels, url)
        delete(um.monitors, url)
        return nil
    }
    return fmt.Errorf("URL %s is not being monitored", url)
}

func (um *UptimeMonitor) monitorURL(url string, interval time.Duration, stop chan struct{}) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-stop:
            return
        case <-ticker.C:
            um.checkURL(url)
        }
    }
}

func (um *UptimeMonitor) checkURL(url string) {
    start := time.Now()
    resp, err := um.client.Get(url)
    responseTime := time.Since(start).Milliseconds()

    entry := LogEntry{
        Timestamp:    time.Now(),
        URL:          url,
        ResponseTime: responseTime,
    }

    if err != nil {
        entry.Success = false
        entry.Error = err.Error()
        um.handleFailure(entry)
        return
    }
    defer resp.Body.Close()

    entry.StatusCode = resp.StatusCode
    entry.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

    um.mu.Lock()
    um.logs = append(um.logs, entry)
    um.mu.Unlock()

    if !entry.Success {
        um.handleFailure(entry)
    } else {
        um.handleSuccess(url)
    }
}

func (um *UptimeMonitor) handleFailure(entry LogEntry) {
    um.mu.Lock()
    defer um.mu.Unlock()

    um.logs = append(um.logs, entry)
    
    // Check if there's an ongoing downtime
    lastDowntime := um.getLastDowntime(entry.URL)
    if lastDowntime == nil || !lastDowntime.EndTime.IsZero() {
        // Start new downtime
        um.downtimes = append(um.downtimes, DowntimeEntry{
            URL:         entry.URL,
            StartTime:   entry.Timestamp,
            StatusCode:  entry.StatusCode,
            ErrorDetail: entry.Error,
        })
    }
}

func (um *UptimeMonitor) handleSuccess(url string) {
    um.mu.Lock()
    defer um.mu.Unlock()

    lastDowntime := um.getLastDowntime(url)
    if lastDowntime != nil && lastDowntime.EndTime.IsZero() {
        lastDowntime.EndTime = time.Now()
        lastDowntime.Duration = lastDowntime.EndTime.Sub(lastDowntime.StartTime).String()
    }
}

func (um *UptimeMonitor) getLastDowntime(url string) *DowntimeEntry {
    for i := len(um.downtimes) - 1; i >= 0; i-- {
        if um.downtimes[i].URL == url {
            return &um.downtimes[i]
        }
    }
    return nil
}

func (um *UptimeMonitor) GetLogs(url string) []LogEntry {
    um.mu.RLock()
    defer um.mu.RUnlock()

    var urlLogs []LogEntry
    for _, log := range um.logs {
        if log.URL == url {
            urlLogs = append(urlLogs, log)
        }
    }
    return urlLogs
}

func (um *UptimeMonitor) GetDowntimes(url string) []DowntimeEntry {
    um.mu.RLock()
    defer um.mu.RUnlock()

    var urlDowntimes []DowntimeEntry
    for _, downtime := range um.downtimes {
        if downtime.URL == url {
            urlDowntimes = append(urlDowntimes, downtime)
        }
    }
    return urlDowntimes
}

// HTTP handlers
func (um *UptimeMonitor) HandleAddMonitor(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        URL      string `json:"url"`
        Interval int    `json:"interval,omitempty"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    interval := time.Duration(req.Interval) * time.Second
    if err := um.AddMonitor(req.URL, interval); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusCreated)
}

func (um *UptimeMonitor) HandleRemoveMonitor(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    url := r.URL.Query().Get("url")
    if url == "" {
        http.Error(w, "URL parameter is required", http.StatusBadRequest)
        return
    }

    if err := um.RemoveMonitor(url); err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func (um *UptimeMonitor) HandleGetLogs(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    url := r.URL.Query().Get("url")
    if url == "" {
        http.Error(w, "URL parameter is required", http.StatusBadRequest)
        return
    }

    logs := um.GetLogs(url)
    json.NewEncoder(w).Encode(logs)
}

func (um *UptimeMonitor) HandleGetDowntimes(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    url := r.URL.Query().Get("url")
    if url == "" {
        http.Error(w, "URL parameter is required", http.StatusBadRequest)
        return
    }

    downtimes := um.GetDowntimes(url)
    json.NewEncoder(w).Encode(downtimes)
}