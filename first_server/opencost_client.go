package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// ===== Structs =====

type CloudCost struct {
	Name      string  `json:"name"`
	CPUCost   float64 `json:"cpuCost"`
	GPUCost   float64 `json:"gpuCost"`
	TotalCost float64 `json:"totalCost"`
}

type Allocation struct {
	Namespace  string  `json:"namespace"`
	ResourceID string  `json:"resource_id"`
	CPUCost    float64 `json:"cpu_cost"`
	MemoryCost float64 `json:"memory_cost"`
	GPUCost    float64 `json:"gpu_cost"`
	TotalCost  float64 `json:"total_cost"`
	StartTime  string  `json:"start_time"`
	EndTime    string  `json:"end_time"`
}

type Asset struct {
	AssetID  string  `json:"asset_id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Status   string  `json:"status"`
	Provider string  `json:"provider"`
	Region   string  `json:"region"`
	Cost     float64 `json:"cost"`
}

// ===== Filter-enabled fetch functions =====

// CloudCosts: optional "namespace" filter (we treat matching by VM/pod name for now)
func getCloudCostsWithFilters(namespace string) ([]CloudCost, string, error) {
	baseURL := "http://localhost:9005/cloudCosts"
	params := []string{}
	if namespace != "" {
		params = append(params, "namespace="+url.QueryEscape(namespace))
	}
	if len(params) > 0 {
		baseURL += "?" + strings.Join(params, "&")
	}

	log.Printf("[MCP Client] Fetching URL: %s\n", baseURL)

	resp, err := http.Get(baseURL)
	if err != nil {
		return nil, baseURL, fmt.Errorf("failed to fetch cloud costs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, baseURL, fmt.Errorf("error %d: %s", resp.StatusCode, string(body))
	}

	var data []CloudCost
	err = json.NewDecoder(resp.Body).Decode(&data)
	return data, baseURL, err
}

// Allocations: filters for namespace, start, end
func getAllocationsWithFilters(namespace, start, end string) ([]Allocation, string, error) {
	baseURL := "http://localhost:9005/allocations"
	params := []string{}
	if namespace != "" {
		params = append(params, "namespace="+url.QueryEscape(namespace))
	}
	if start != "" {
		params = append(params, "start="+url.QueryEscape(start))
	}
	if end != "" {
		params = append(params, "end="+url.QueryEscape(end))
	}
	if len(params) > 0 {
		baseURL += "?" + strings.Join(params, "&")
	}

	log.Printf("[MCP Client] Fetching URL: %s\n", baseURL)

	resp, err := http.Get(baseURL)
	if err != nil {
		return nil, baseURL, fmt.Errorf("failed to fetch allocations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, baseURL, fmt.Errorf("error %d: %s", resp.StatusCode, string(body))
	}

	var data []Allocation
	err = json.NewDecoder(resp.Body).Decode(&data)
	return data, baseURL, err
}

// Assets: filters for provider and region
func getAssetsWithFilters(provider, region string) ([]Asset, string, error) {
	baseURL := "http://localhost:9005/assets"
	params := []string{}
	if provider != "" {
		params = append(params, "provider="+url.QueryEscape(provider))
	}
	if region != "" {
		params = append(params, "region="+url.QueryEscape(region))
	}
	if len(params) > 0 {
		baseURL += "?" + strings.Join(params, "&")
	}

	log.Printf("[MCP Client] Fetching URL: %s\n", baseURL)

	resp, err := http.Get(baseURL)
	if err != nil {
		return nil, baseURL, fmt.Errorf("failed to fetch assets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, baseURL, fmt.Errorf("error %d: %s", resp.StatusCode, string(body))
	}

	var data []Asset
	err = json.NewDecoder(resp.Body).Decode(&data)
	return data, baseURL, err
}
