package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/vpsie/vpsie-loadbalancer/pkg/models"
)

const (
	// maxResponseSize limits API response body size to prevent DoS attacks
	maxResponseSize = 10 * 1024 * 1024 // 10MB

	// httpsScheme is the HTTPS URL scheme
	httpsScheme = "https"
	// httpScheme is the HTTP URL scheme
	httpScheme = "http"
)

// VPSieClient handles communication with the VPSie API
type VPSieClient struct {
	httpClient     *http.Client
	apiKey         string
	baseURL        string
	loadBalancerID string
}

// isPrivateOrLocalhost checks if an IP or hostname is private or localhost
func isPrivateOrLocalhost(host string) bool {
	// Check for localhost
	if host == "localhost" || strings.HasPrefix(host, "127.") || host == "::1" {
		return true
	}

	// Parse as IP
	ip := net.ParseIP(host)
	if ip == nil {
		// Not an IP, could be hostname - resolve it
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return false
		}
		ip = ips[0]
	}

	// Check for private IP ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16", // AWS metadata
		"fd00::/8",       // IPv6 ULA
		"fe80::/10",      // IPv6 link-local
	}

	for _, cidr := range privateRanges {
		_, ipNet, parseErr := net.ParseCIDR(cidr)
		if parseErr != nil {
			// This should never happen with hardcoded CIDRs, but check anyway
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// TestMode allows tests to bypass hostname validation. Must only be set in test code.
var TestMode bool

// validateHostname checks if the hostname is allowed
func validateHostname(hostname string) error {
	// Allow bypassing validation in test mode only
	if TestMode {
		return nil
	}

	// For production URLs, check against whitelist and reject private IPs
	if isPrivateOrLocalhost(hostname) {
		return fmt.Errorf("base URL must not be localhost or private IP address")
	}

	allowedDomains := []string{"api.vpsie.com", "vpsie.com"}
	for _, domain := range allowedDomains {
		if hostname == domain || strings.HasSuffix(hostname, "."+domain) {
			return nil
		}
	}

	return fmt.Errorf("base URL domain not in allowed list: %s", hostname)
}

// idPattern matches valid resource IDs (alphanumeric, hyphens, underscores)
var idPattern = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

// sanitizeID validates and escapes a resource ID for safe use in URL paths
func sanitizeID(id string) string {
	if !idPattern.MatchString(id) {
		return url.PathEscape(id)
	}
	return id
}

// NewVPSieClient creates a new VPSie API client with URL validation
func NewVPSieClient(apiKey, baseURL, loadBalancerID string) (*VPSieClient, error) {
	// Validate base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Only allow HTTPS (or HTTP for local development)
	if parsedURL.Scheme != httpsScheme && parsedURL.Scheme != httpScheme {
		return nil, fmt.Errorf("base URL must use HTTP or HTTPS scheme")
	}

	// Validate hostname matches expected VPSie domains (whitelist)
	if hostErr := validateHostname(parsedURL.Hostname()); hostErr != nil {
		return nil, hostErr
	}

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
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Limit maximum redirects to 3
				if len(via) >= 3 {
					return fmt.Errorf("stopped after 3 redirects")
				}
				// Ensure redirect stays on the same host (prevent open redirect)
				if req.URL.Host != via[0].URL.Host {
					return fmt.Errorf("redirect to different host not allowed: %s -> %s", via[0].URL.Host, req.URL.Host)
				}
				// Ensure redirect maintains HTTPS if original was HTTPS
				if via[0].URL.Scheme == httpsScheme && req.URL.Scheme != httpsScheme {
					return fmt.Errorf("redirect from HTTPS to HTTP not allowed")
				}
				return nil
			},
		},
	}, nil
}

// truncateErrorMessage truncates error messages to prevent sensitive information disclosure
func truncateErrorMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen] + "... (truncated)"
}

// doWithRetry retries a function on 5xx responses and network errors with exponential backoff
func doWithRetry(fn func() (*http.Response, error), maxRetries int) (*http.Response, error) {
	var resp *http.Response
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = fn()
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		if attempt < maxRetries {
			// Close body from failed attempt before retry
			if resp != nil {
				resp.Body.Close()
			}
			backoff := time.Duration(1<<uint(attempt)) * time.Second // 1s, 2s, 4s
			time.Sleep(backoff)
		}
	}
	return resp, err
}

// GetLoadBalancerConfig fetches the load balancer configuration from VPSie API
func (c *VPSieClient) GetLoadBalancerConfig(ctx context.Context) (*models.LoadBalancer, error) {
	// Add timeout to prevent hanging requests
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/loadbalancers/%s", c.baseURL, sanitizeID(c.loadBalancerID))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := doWithRetry(func() (*http.Response, error) {
		return c.httpClient.Do(req)
	}, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		// Drain response body to enable HTTP connection reuse
		//nolint:errcheck // Intentionally ignore - draining is best effort for connection reuse
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
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

	url := fmt.Sprintf("%s/loadbalancers/%s/status", c.baseURL, sanitizeID(c.loadBalancerID))

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
		//nolint:errcheck // Intentionally ignore - draining is best effort for connection reuse
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
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

	url := fmt.Sprintf("%s/loadbalancers/%s/backends/%s/health", c.baseURL, sanitizeID(c.loadBalancerID), sanitizeID(backendID))

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
		//nolint:errcheck // Intentionally ignore - draining is best effort for connection reuse
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
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

	url := fmt.Sprintf("%s/loadbalancers/%s/metrics", c.baseURL, sanitizeID(c.loadBalancerID))

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
		//nolint:errcheck // Intentionally ignore - draining is best effort for connection reuse
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
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

	url := fmt.Sprintf("%s/loadbalancers/%s/events", c.baseURL, sanitizeID(c.loadBalancerID))

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
		//nolint:errcheck // Intentionally ignore - draining is best effort for connection reuse
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
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
