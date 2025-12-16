package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vpsie/vpsie-loadbalancer/pkg/models"
)

const (
	// maxResponseSize limits API response body size to prevent DoS attacks
	maxResponseSize = 10 * 1024 * 1024 // 10MB
)

// VPSieClient handles communication with the VPSie API
type VPSieClient struct {
	httpClient     *http.Client
	apiKey         string
	baseURL        string
	loadBalancerID string
}

// NewVPSieClient creates a new VPSie API client
func NewVPSieClient(apiKey, baseURL, loadBalancerID string) *VPSieClient {
	return &VPSieClient{
		apiKey:         apiKey,
		baseURL:        baseURL,
		loadBalancerID: loadBalancerID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// truncateErrorMessage truncates error messages to prevent sensitive information disclosure
func truncateErrorMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen] + "... (truncated)"
}

// GetLoadBalancerConfig fetches the load balancer configuration from VPSie API
func (c *VPSieClient) GetLoadBalancerConfig(ctx context.Context) (*models.LoadBalancer, error) {
	// Add timeout to prevent hanging requests
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/loadbalancers/%s", c.baseURL, c.loadBalancerID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		// Drain response body to enable HTTP connection reuse
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		if readErr != nil {
			return nil, fmt.Errorf("API returned status %d (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		errMsg := truncateErrorMessage(string(body), 200)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, errMsg)
	}

	var lb models.LoadBalancer
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	if decodeErr := json.NewDecoder(limitedReader).Decode(&lb); decodeErr != nil {
		return nil, fmt.Errorf("failed to decode response: %w", decodeErr)
	}

	return &lb, nil
}

// UpdateLoadBalancerStatus updates the load balancer status in VPSie
func (c *VPSieClient) UpdateLoadBalancerStatus(ctx context.Context, status string) error {
	// Add timeout to prevent hanging requests
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/loadbalancers/%s/status", c.baseURL, c.loadBalancerID)

	payload := map[string]string{
		"status": status,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		// Drain response body to enable HTTP connection reuse
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		if readErr != nil {
			return fmt.Errorf("API returned status %d (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		errMsg := truncateErrorMessage(string(body), 200)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, errMsg)
	}

	return nil
}

// UpdateBackendStatus updates the status of a specific backend server
func (c *VPSieClient) UpdateBackendStatus(ctx context.Context, backendID string, healthy bool) error {
	// Add timeout to prevent hanging requests
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/loadbalancers/%s/backends/%s/health", c.baseURL, c.loadBalancerID, backendID)

	status := "unhealthy"
	if healthy {
		status = "healthy"
	}

	payload := map[string]string{
		"status": status,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal backend status: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		// Drain response body to enable HTTP connection reuse
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		if readErr != nil {
			return fmt.Errorf("API returned status %d (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		errMsg := truncateErrorMessage(string(body), 200)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, errMsg)
	}

	return nil
}

// ReportMetrics sends metrics data to VPSie API
func (c *VPSieClient) ReportMetrics(ctx context.Context, metrics map[string]interface{}) error {
	// Add timeout to prevent hanging requests
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/loadbalancers/%s/metrics", c.baseURL, c.loadBalancerID)

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		// Drain response body to enable HTTP connection reuse
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		if readErr != nil {
			return fmt.Errorf("API returned status %d (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		errMsg := truncateErrorMessage(string(body), 200)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, errMsg)
	}

	return nil
}

// SendEvent sends an event notification to VPSie API
func (c *VPSieClient) SendEvent(ctx context.Context, eventType, message string, metadata map[string]interface{}) error {
	// Add timeout to prevent hanging requests
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/loadbalancers/%s/events", c.baseURL, c.loadBalancerID)

	payload := map[string]interface{}{
		"type":      eventType,
		"message":   message,
		"metadata":  metadata,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		// Drain response body to enable HTTP connection reuse
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		if readErr != nil {
			return fmt.Errorf("API returned status %d (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		errMsg := truncateErrorMessage(string(body), 200)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, errMsg)
	}

	return nil
}
