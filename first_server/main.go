package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// AgenticQuery represents a flexible query structure that supports both natural language queries
// and structured filters. It also holds context information to support multi-turn conversations,
// making the API more AI/agent-friendly.
type AgenticQuery struct {
	Query   string `json:"query,omitempty"` // The natural language query, e.g. "Show prod costs"
	Filters struct {
		Namespace string `json:"namespace,omitempty"`
		Start     string `json:"start,omitempty"`
		End       string `json:"end,omitempty"`
		Provider  string `json:"provider,omitempty"`
		Region    string `json:"region,omitempty"`
	} `json:"filters,omitempty"`
	Context struct {
		SessionID           string   `json:"session_id,omitempty"`           // Session identifier for conversation tracking
		PreviousQuery       string   `json:"previous_query,omitempty"`       // Last query made in this session
		ConversationContext []string `json:"conversation_context,omitempty"` // Full history of queries in this session
	} `json:"context,omitempty"`
}

// sessions stores query histories per session to enable multi-turn conversational context.
var sessions = make(map[string][]string)

// parseDate safely parses an RFC3339 timestamp string. Returns zero time if empty.
func parseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, dateStr)
}

// cloudCostsHandler handles GET and POST requests to /cloudCosts.
// Supports filter parameters and conversation context for AI readiness.
func cloudCostsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[MCP] /cloudCosts request received")

	// Initialize filters with GET query params
	namespace := r.URL.Query().Get("namespace")
	sessionID := ""
	queryText := ""
	previous := ""
	history := []string{}

	if r.Method == http.MethodPost {
		// Decode AgenticQuery JSON body if POST
		var aq AgenticQuery
		if err := json.NewDecoder(r.Body).Decode(&aq); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		// Override filters and context from POST body
		namespace = aq.Filters.Namespace
		sessionID = aq.Context.SessionID
		queryText = aq.Query

		// Update conversation history in memory
		if sessionID != "" && queryText != "" {
			if existing, ok := sessions[sessionID]; ok && len(existing) > 0 {
				previous = existing[len(existing)-1]
				history = append(existing, queryText)
			} else {
				history = []string{queryText}
			}
			sessions[sessionID] = history
		}
		log.Printf("[MCP] Parsed agentic POST query: %+v\n", aq)
	}

	// Fetch data from downstream (mock server or real backend)
	data, url, err := getCloudCostsWithFilters(namespace)
	if err != nil {
		http.Error(w, "Failed to get cloud costs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[MCP Client] Fetching URL: %s\n", url)
	log.Printf("[MCP] /cloudCosts — received %d records\n", len(data))

	// Apply additional local filtering to be safe
	filtered := []CloudCost{}
	if namespace != "" {
		for _, cost := range data {
			if strings.Contains(strings.ToLower(cost.Name), strings.ToLower(namespace)) {
				filtered = append(filtered, cost)
			}
		}
	} else {
		filtered = data
	}

	// Compose response including data, filters used, and conversation context
	resp := map[string]interface{}{
		"data": filtered,
		"meta": map[string]interface{}{
			"filtersUsed":          map[string]string{"namespace": namespace},
			"session_id":           sessionID,
			"previous_query":       previous,
			"conversation_context": history,
			"total":                len(filtered),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// allocationsHandler handles GET and POST requests to /allocations.
// Supports filtering by namespace and time range, and tracks session context.
func allocationsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[MCP] /allocations request received")

	namespace := r.URL.Query().Get("namespace")
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	sessionID := ""
	queryText := ""
	previous := ""
	history := []string{}

	if r.Method == http.MethodPost {
		var aq AgenticQuery
		if err := json.NewDecoder(r.Body).Decode(&aq); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		namespace = aq.Filters.Namespace
		start = aq.Filters.Start
		end = aq.Filters.End
		sessionID = aq.Context.SessionID
		queryText = aq.Query

		if sessionID != "" && queryText != "" {
			if existing, ok := sessions[sessionID]; ok && len(existing) > 0 {
				previous = existing[len(existing)-1]
				history = append(existing, queryText)
			} else {
				history = []string{queryText}
			}
			sessions[sessionID] = history
		}
		log.Printf("[MCP] Parsed agentic POST query: %+v\n", aq)
	}

	// Fetch data from downstream source
	data, url, err := getAllocationsWithFilters(namespace, start, end)
	if err != nil {
		http.Error(w, "Failed to get allocations: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[MCP Client] Fetching URL: %s\n", url)
	log.Printf("[MCP] /allocations — received %d records\n", len(data))

	// Filter results locally by namespace and time range
	startTime, _ := parseDate(start)
	endTime, _ := parseDate(end)
	filtered := []Allocation{}
	for _, alloc := range data {
		if namespace != "" && alloc.Namespace != namespace {
			continue
		}
		allocStart, _ := time.Parse(time.RFC3339, alloc.StartTime)
		allocEnd, _ := time.Parse(time.RFC3339, alloc.EndTime)
		if !startTime.IsZero() && allocEnd.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && allocStart.After(endTime) {
			continue
		}
		filtered = append(filtered, alloc)
	}

	resp := map[string]interface{}{
		"data": filtered,
		"meta": map[string]interface{}{
			"filtersUsed":          map[string]string{"namespace": namespace, "start": start, "end": end},
			"session_id":           sessionID,
			"previous_query":       previous,
			"conversation_context": history,
			"total":                len(filtered),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// assetsHandler handles GET and POST requests to /assets.
// Supports filtering by provider and region, with session context tracking.
func assetsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[MCP] /assets request received")

	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")
	sessionID := ""
	queryText := ""
	previous := ""
	history := []string{}

	if r.Method == http.MethodPost {
		var aq AgenticQuery
		if err := json.NewDecoder(r.Body).Decode(&aq); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		// Fallbacks for filters to handle different client usages
		if aq.Filters.Provider != "" {
			provider = aq.Filters.Provider
		} else if aq.Filters.Namespace != "" {
			provider = aq.Filters.Namespace
		}
		if aq.Filters.Region != "" {
			region = aq.Filters.Region
		} else if aq.Filters.Start != "" {
			region = aq.Filters.Start
		}
		sessionID = aq.Context.SessionID
		queryText = aq.Query

		if sessionID != "" && queryText != "" {
			if existing, ok := sessions[sessionID]; ok && len(existing) > 0 {
				previous = existing[len(existing)-1]
				history = append(existing, queryText)
			} else {
				history = []string{queryText}
			}
			sessions[sessionID] = history
		}
		log.Printf("[MCP] Parsed agentic POST query: %+v\n", aq)
	}

	data, url, err := getAssetsWithFilters(provider, region)
	if err != nil {
		http.Error(w, "Failed to get assets: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[MCP Client] Fetching URL: %s\n", url)
	log.Printf("[MCP] /assets — received %d records\n", len(data))

	filtered := []Asset{}
	for _, asset := range data {
		if provider != "" && !strings.EqualFold(asset.Provider, provider) {
			continue
		}
		if region != "" && !strings.EqualFold(asset.Region, region) {
			continue
		}
		filtered = append(filtered, asset)
	}

	resp := map[string]interface{}{
		"data": filtered,
		"meta": map[string]interface{}{
			"filtersUsed":          map[string]string{"provider": provider, "region": region},
			"session_id":           sessionID,
			"previous_query":       previous,
			"conversation_context": history,
			"total":                len(filtered),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	// Register HTTP handlers for MCP endpoints
	http.HandleFunc("/cloudCosts", cloudCostsHandler)
	http.HandleFunc("/allocations", allocationsHandler)
	http.HandleFunc("/assets", assetsHandler)

	log.Println("Starting MCP server on :9004...")
	if err := http.ListenAndServe(":9004", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
