package main

import (
	"log"
	"net/http"
	"urlmonitor/src/entity"
)

func main() {
	monitor := entity.NewUptimeMonitor()

	// API endpoints
	http.HandleFunc("/monitor/add", monitor.HandleAddMonitor)
	http.HandleFunc("/monitor/remove", monitor.HandleRemoveMonitor)
	http.HandleFunc("/monitor/logs", monitor.HandleGetLogs)
	http.HandleFunc("/monitor/downtimes", monitor.HandleGetDowntimes)

	log.Printf("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
