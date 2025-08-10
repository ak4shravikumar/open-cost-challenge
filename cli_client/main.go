package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// ----- Payload Structs -----
// Filters: endpoint-specific parameters
type Filters struct {
	Namespace string `json:"namespace,omitempty"`
	Start     string `json:"start,omitempty"`
	End       string `json:"end,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Region    string `json:"region,omitempty"`
}

// Context: session_id used for conversation tracking
type Context struct {
	SessionID string `json:"session_id,omitempty"`
}

// AgenticQuery: full request payload
type AgenticQuery struct {
	Query   string  `json:"query,omitempty"`
	Filters Filters `json:"filters,omitempty"`
	Context Context `json:"context,omitempty"`
}

func main() {
	// --- Graceful exit handler for Ctrl+C ---
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("\n\nReceived Ctrl+C — exiting MCP CLI cleanly. Goodbye!")
		os.Exit(0)
	}()

	// --- CLI setup ---
	reader := bufio.NewReader(os.Stdin)
	sessionID := "cli-demo-001" // fixed session; can be changed for testing

	fmt.Println("MCP CLI Conversation Client")
	fmt.Println("Supports: allocations, cloudCosts, assets")
	fmt.Println("Type 'quit' or 'exit' as the query to end session.\n")

	// --- Main interactive loop ---
	for {
		// 1️⃣ Choose endpoint
		fmt.Print("Choose endpoint (allocations/cloudCosts/assets): ")
		endpoint, _ := reader.ReadString('\n')
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			endpoint = "allocations" // default if empty
		}

		// 2️⃣ Enter natural language query
		fmt.Print("Enter query: ")
		query, _ := reader.ReadString('\n')
		query = strings.TrimSpace(query)
		if q := strings.ToLower(query); q == "quit" || q == "exit" {
			fmt.Println("\nGoodbye! MCP CLI session ended.")
			break
		}

		// 3️⃣ Endpoint-specific filter prompts
		var namespace, start, end, provider, region string
		switch endpoint {
		case "allocations":
			fmt.Print("Namespace: ")
			namespace, _ = reader.ReadString('\n')
			namespace = strings.TrimSpace(namespace)

			fmt.Print("Start date (RFC3339): ")
			start, _ = reader.ReadString('\n')
			start = strings.TrimSpace(start)

			fmt.Print("End date (RFC3339): ")
			end, _ = reader.ReadString('\n')
			end = strings.TrimSpace(end)

		case "cloudCosts":
			fmt.Print("Namespace: ")
			namespace, _ = reader.ReadString('\n')
			namespace = strings.TrimSpace(namespace)

		case "assets":
			fmt.Print("Provider: ")
			provider, _ = reader.ReadString('\n')
			provider = strings.TrimSpace(provider)

			fmt.Print("Region: ")
			region, _ = reader.ReadString('\n')
			region = strings.TrimSpace(region)
		}

		// 4️⃣ Build agentic JSON payload
		aq := AgenticQuery{
			Query: query,
			Filters: Filters{
				Namespace: namespace,
				Start:     start,
				End:       end,
				Provider:  provider,
				Region:    region,
			},
			Context: Context{
				SessionID: sessionID,
			},
		}
		payload, _ := json.Marshal(aq)

		// 5️⃣ Send POST to MCP server
		url := fmt.Sprintf("http://localhost:9004/%s", endpoint)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			fmt.Println("Error sending request:", err)
			continue
		}
		defer resp.Body.Close()

		// 6️⃣ Decode JSON response
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			fmt.Println("Error decoding response:", err)
			continue
		}

		// 7️⃣ Print metadata (conversation context info)
		fmt.Println("\n--- MCP Response ---")
		if meta, ok := result["meta"].(map[string]interface{}); ok {
			fmt.Println("Session ID:          ", meta["session_id"])
			fmt.Println("Previous Query:      ", meta["previous_query"])
			fmt.Println("Conversation Context:", meta["conversation_context"])
			fmt.Println("Total Records:       ", meta["total"])
		}

		// 8️⃣ Pretty print data records
		dataArray, ok := result["data"].([]interface{})
		if !ok || len(dataArray) == 0 {
			fmt.Println("\n(No data records returned.)\n")
			continue
		}

		fmt.Println("\n--- Data Records ---")
		switch endpoint {
		case "allocations":
			fmt.Printf("%-12s %-12s %-8s %-8s %-8s %-8s\n",
				"Namespace", "ResID", "CPU", "Memory", "GPU", "Total")
			fmt.Println(strings.Repeat("-", 60))
			for _, item := range dataArray {
				rec := item.(map[string]interface{})
				fmt.Printf("%-12v %-12v %-8.2f %-8.2f %-8.2f %-8.2f\n",
					rec["namespace"], rec["resource_id"],
					rec["cpu_cost"], rec["memory_cost"], rec["gpu_cost"], rec["total_cost"])
			}

		case "cloudCosts":
			fmt.Printf("%-20s %-10s\n", "Name", "Cost")
			fmt.Println(strings.Repeat("-", 30))
			for _, item := range dataArray {
				rec := item.(map[string]interface{})
				fmt.Printf("%-20v %-10.2f\n", rec["name"], rec["cost"])
			}

		case "assets":
			fmt.Printf("%-10s %-12s %-15s %-10s\n",
				"Provider", "Region", "Name", "Type")
			fmt.Println(strings.Repeat("-", 50))
			for _, item := range dataArray {
				rec := item.(map[string]interface{})
				fmt.Printf("%-10v %-12v %-15v %-10v\n",
					rec["provider"], rec["region"], rec["name"], rec["type"])
			}
		}
		fmt.Println()
	}
}
