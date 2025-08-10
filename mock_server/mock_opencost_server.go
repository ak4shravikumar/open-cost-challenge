package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// ===== Hardcoded Data =====

var cloudCostsData = []map[string]interface{}{
	{"name": "prod-vm-1", "cpuCost": 10.5, "gpuCost": 5.0, "totalCost": 15.5},
	{"name": "dev-vm-2", "cpuCost": 8.0, "gpuCost": 3.5, "totalCost": 11.5},
}

var allocationsData = []map[string]interface{}{
	{
		"namespace":   "dev",
		"resource_id": "pod-123",
		"cpu_cost":    4.5,
		"memory_cost": 1.2,
		"gpu_cost":    0,
		"total_cost":  5.7,
		"start_time":  "2025-08-01T00:00:00Z",
		"end_time":    "2025-08-02T00:00:00Z",
	},
	{
		"namespace":   "prod",
		"resource_id": "pod-456",
		"cpu_cost":    10,
		"memory_cost": 3.5,
		"gpu_cost":    0,
		"total_cost":  13.5,
		"start_time":  "2025-08-01T00:00:00Z",
		"end_time":    "2025-08-02T00:00:00Z",
	},
}

var assetsData = []map[string]interface{}{
	{
		"asset_id": "asset-001",
		"name":     "AWS EC2 m5.large",
		"type":     "VM",
		"status":   "active",
		"provider": "AWS",
		"region":   "us-west-2",
		"cost":     120.5,
	},
	{
		"asset_id": "asset-002",
		"name":     "Azure SQL Database",
		"type":     "Database",
		"status":   "active",
		"provider": "Azure",
		"region":   "centralindia",
		"cost":     300.75,
	},
}

// ===== Handlers with Filtering =====

// /cloudCosts
func cloudCostsHandler(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	filtered := []map[string]interface{}{}
	for _, cost := range cloudCostsData {
		if namespace == "" ||
			strings.Contains(strings.ToLower(cost["name"].(string)), strings.ToLower(namespace)) {
			filtered = append(filtered, cost)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

// /allocations
func allocationsHandler(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	startTime, errStart := time.Parse(time.RFC3339, start)
	endTime, errEnd := time.Parse(time.RFC3339, end)

	filtered := []map[string]interface{}{}
	for _, alloc := range allocationsData {
		// Namespace filter
		if namespace != "" && alloc["namespace"] != namespace {
			continue
		}
		// Date range filter
		if start != "" && errStart == nil {
			allocEnd, err := time.Parse(time.RFC3339, alloc["end_time"].(string))
			if err == nil && allocEnd.Before(startTime) {
				continue
			}
		}
		if end != "" && errEnd == nil {
			allocStart, err := time.Parse(time.RFC3339, alloc["start_time"].(string))
			if err == nil && allocStart.After(endTime) {
				continue
			}
		}
		filtered = append(filtered, alloc)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

// /assets
func assetsHandler(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")
	filtered := []map[string]interface{}{}
	for _, asset := range assetsData {
		if provider != "" && !strings.EqualFold(asset["provider"].(string), provider) {
			continue
		}
		if region != "" && !strings.EqualFold(asset["region"].(string), region) {
			continue
		}
		filtered = append(filtered, asset)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

func main() {
	http.HandleFunc("/cloudCosts", cloudCostsHandler)
	http.HandleFunc("/allocations", allocationsHandler)
	http.HandleFunc("/assets", assetsHandler)

	log.Println("Mock OpenCost server running on :9005")
	if err := http.ListenAndServe(":9005", nil); err != nil {
		log.Fatalf("Mock server failed to start: %v", err)
	}
}
