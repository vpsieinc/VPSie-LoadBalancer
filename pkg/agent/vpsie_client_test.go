package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/vpsie/vpsie-loadbalancer/pkg/models"
)

func TestMain(m *testing.M) {
	TestMode = true
	os.Exit(m.Run())
}

func TestNewVPSieClient(t *testing.T) {
	client, _ := NewVPSieClient("test-key", "https://api.test.com", "lb-123")

	if client.apiKey != "test-key" {
		t.Errorf("apiKey = %v, want test-key", client.apiKey)
	}
	if client.baseURL != "https://api.test.com" {
		t.Errorf("baseURL = %v, want https://api.test.com", client.baseURL)
	}
	if client.loadBalancerID != "lb-123" {
		t.Errorf("loadBalancerID = %v, want lb-123", client.loadBalancerID)
	}
	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestVPSieClient_GetLoadBalancerConfig(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		lb := &models.LoadBalancer{
			ID:        "lb-123",
			Name:      "test-lb",
			Protocol:  models.ProtocolHTTP,
			Algorithm: models.AlgoRoundRobin,
			Port:      80,
			Backends: []models.Backend{
				{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("Expected GET request, got %s", r.Method)
			}
			if r.URL.Path != "/loadbalancers/lb-123" {
				t.Errorf("Expected path /loadbalancers/lb-123, got %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Error("Authorization header not set correctly")
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(lb)
		}))
		defer server.Close()

		client, _ := NewVPSieClient("test-key", server.URL, "lb-123")
		result, err := client.GetLoadBalancerConfig(context.Background())

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result.ID != "lb-123" {
			t.Errorf("Expected ID lb-123, got %s", result.ID)
		}
	})

	t.Run("API error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("load balancer not found"))
		}))
		defer server.Close()

		client, _ := NewVPSieClient("test-key", server.URL, "lb-123")
		_, err := client.GetLoadBalancerConfig(context.Background())

		if err == nil {
			t.Error("Expected error for 404 response")
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		client, _ := NewVPSieClient("test-key", server.URL, "lb-123")
		_, err := client.GetLoadBalancerConfig(context.Background())

		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestVPSieClient_UpdateLoadBalancerStatus(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PUT" {
				t.Errorf("Expected PUT request, got %s", r.Method)
			}
			if r.URL.Path != "/loadbalancers/lb-123/status" {
				t.Errorf("Expected path /loadbalancers/lb-123/status, got %s", r.URL.Path)
			}

			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["status"] != "active" {
				t.Errorf("Expected status 'active', got %s", payload["status"])
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, _ := NewVPSieClient("test-key", server.URL, "lb-123")
		err := client.UpdateLoadBalancerStatus(context.Background(), "active")

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal server error"))
		}))
		defer server.Close()

		client, _ := NewVPSieClient("test-key", server.URL, "lb-123")
		err := client.UpdateLoadBalancerStatus(context.Background(), "active")

		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})
}

func TestVPSieClient_UpdateBackendStatus(t *testing.T) {
	t.Run("update backend to healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PUT" {
				t.Errorf("Expected PUT request, got %s", r.Method)
			}
			if r.URL.Path != "/loadbalancers/lb-123/backends/be-1/health" {
				t.Errorf("Unexpected path: %s", r.URL.Path)
			}

			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["status"] != "healthy" {
				t.Errorf("Expected status 'healthy', got %s", payload["status"])
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, _ := NewVPSieClient("test-key", server.URL, "lb-123")
		err := client.UpdateBackendStatus(context.Background(), "be-1", true)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("update backend to unhealthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["status"] != "unhealthy" {
				t.Errorf("Expected status 'unhealthy', got %s", payload["status"])
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, _ := NewVPSieClient("test-key", server.URL, "lb-123")
		err := client.UpdateBackendStatus(context.Background(), "be-1", false)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestVPSieClient_ReportMetrics(t *testing.T) {
	t.Run("successful metrics report", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST request, got %s", r.Method)
			}
			if r.URL.Path != "/loadbalancers/lb-123/metrics" {
				t.Errorf("Unexpected path: %s", r.URL.Path)
			}

			var metrics map[string]interface{}
			json.NewDecoder(r.Body).Decode(&metrics)
			if metrics["connections"] != float64(100) {
				t.Errorf("Expected connections 100, got %v", metrics["connections"])
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, _ := NewVPSieClient("test-key", server.URL, "lb-123")
		metrics := map[string]interface{}{
			"connections": 100,
			"requests":    1000,
		}
		err := client.ReportMetrics(context.Background(), metrics)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestVPSieClient_SendEvent(t *testing.T) {
	t.Run("successful event send", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST request, got %s", r.Method)
			}
			if r.URL.Path != "/loadbalancers/lb-123/events" {
				t.Errorf("Unexpected path: %s", r.URL.Path)
			}

			var event map[string]interface{}
			json.NewDecoder(r.Body).Decode(&event)
			if event["type"] != "config_updated" {
				t.Errorf("Expected type 'config_updated', got %v", event["type"])
			}
			if event["message"] != "Config applied" {
				t.Errorf("Expected message 'Config applied', got %v", event["message"])
			}

			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		client, _ := NewVPSieClient("test-key", server.URL, "lb-123")
		metadata := map[string]interface{}{"version": "1.0"}
		err := client.SendEvent(context.Background(), "config_updated", "Config applied", metadata)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
