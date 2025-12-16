package agent

import (
	"testing"
	"time"

	"github.com/vpsie/vpsie-loadbalancer/pkg/models"
)

func TestAgent_computeConfigHash(t *testing.T) {
	agent := &Agent{}

	baseTime := time.Now()
	lb1 := &models.LoadBalancer{
		ID:        "lb-1",
		Name:      "test-lb",
		Protocol:  models.ProtocolHTTP,
		Algorithm: models.AlgoRoundRobin,
		Port:      80,
		Backends: []models.Backend{
			{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
		},
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}

	lb2 := &models.LoadBalancer{
		ID:        "lb-1",
		Name:      "test-lb",
		Protocol:  models.ProtocolHTTP,
		Algorithm: models.AlgoRoundRobin,
		Port:      80,
		Backends: []models.Backend{
			{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
		},
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}

	lb3 := &models.LoadBalancer{
		ID:        "lb-1",
		Name:      "test-lb-modified",
		Protocol:  models.ProtocolHTTP,
		Algorithm: models.AlgoRoundRobin,
		Port:      80,
		Backends: []models.Backend{
			{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
		},
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}

	lb4 := &models.LoadBalancer{
		ID:        "lb-1",
		Name:      "test-lb",
		Protocol:  models.ProtocolHTTP,
		Algorithm: models.AlgoRoundRobin,
		Port:      80,
		Backends: []models.Backend{
			{ID: "be-1", Address: "10.0.0.2", Port: 8080, Enabled: true}, // Different address
		},
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}

	lb5 := &models.LoadBalancer{
		ID:        "lb-1",
		Name:      "test-lb",
		Protocol:  models.ProtocolHTTP,
		Algorithm: models.AlgoRoundRobin,
		Port:      8080, // Different port
		Backends: []models.Backend{
			{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
		},
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}

	t.Run("identical configs produce same hash", func(t *testing.T) {
		hash1 := agent.computeConfigHash(lb1)
		hash2 := agent.computeConfigHash(lb2)

		if hash1 != hash2 {
			t.Errorf("Expected identical configs to have same hash, got %s and %s", hash1, hash2)
		}
	})

	t.Run("different name produces different hash", func(t *testing.T) {
		hash1 := agent.computeConfigHash(lb1)
		hash3 := agent.computeConfigHash(lb3)

		if hash1 == hash3 {
			t.Error("Expected different name to produce different hash")
		}
	})

	t.Run("different backend address produces different hash", func(t *testing.T) {
		hash1 := agent.computeConfigHash(lb1)
		hash4 := agent.computeConfigHash(lb4)

		if hash1 == hash4 {
			t.Error("Expected different backend address to produce different hash")
		}
	})

	t.Run("different port produces different hash", func(t *testing.T) {
		hash1 := agent.computeConfigHash(lb1)
		hash5 := agent.computeConfigHash(lb5)

		if hash1 == hash5 {
			t.Error("Expected different port to produce different hash")
		}
	})

	t.Run("hash is non-empty and reasonable length", func(t *testing.T) {
		hash := agent.computeConfigHash(lb1)

		if hash == "" {
			t.Error("Hash should not be empty")
		}

		// SHA-256 hash in hex should be 64 characters
		if len(hash) != 64 {
			t.Errorf("Hash length = %d, want 64 (SHA-256 hex)", len(hash))
		}
	})

	t.Run("hash changes with backend count", func(t *testing.T) {
		lb6 := &models.LoadBalancer{
			ID:        "lb-1",
			Name:      "test-lb",
			Protocol:  models.ProtocolHTTP,
			Algorithm: models.AlgoRoundRobin,
			Port:      80,
			Backends: []models.Backend{
				{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				{ID: "be-2", Address: "10.0.0.2", Port: 8080, Enabled: true},
			},
			CreatedAt: baseTime,
			UpdatedAt: baseTime,
		}

		hash1 := agent.computeConfigHash(lb1)
		hash6 := agent.computeConfigHash(lb6)

		if hash1 == hash6 {
			t.Error("Expected different backend count to produce different hash")
		}
	})

	t.Run("hash changes with protocol", func(t *testing.T) {
		lb7 := &models.LoadBalancer{
			ID:        "lb-1",
			Name:      "test-lb",
			Protocol:  models.ProtocolTCP, // Different protocol
			Algorithm: models.AlgoRoundRobin,
			Port:      80,
			Backends: []models.Backend{
				{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
			},
			CreatedAt: baseTime,
			UpdatedAt: baseTime,
		}

		hash1 := agent.computeConfigHash(lb1)
		hash7 := agent.computeConfigHash(lb7)

		if hash1 == hash7 {
			t.Error("Expected different protocol to produce different hash")
		}
	})

	t.Run("hash changes with algorithm", func(t *testing.T) {
		lb8 := &models.LoadBalancer{
			ID:        "lb-1",
			Name:      "test-lb",
			Protocol:  models.ProtocolHTTP,
			Algorithm: models.AlgoLeastRequest, // Different algorithm
			Port:      80,
			Backends: []models.Backend{
				{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
			},
			CreatedAt: baseTime,
			UpdatedAt: baseTime,
		}

		hash1 := agent.computeConfigHash(lb1)
		hash8 := agent.computeConfigHash(lb8)

		if hash1 == hash8 {
			t.Error("Expected different algorithm to produce different hash")
		}
	})
}

func TestAgent_IsRunning(t *testing.T) {
	agent := &Agent{}

	if agent.IsRunning() {
		t.Error("Expected agent to not be running initially")
	}

	agent.running.Store(true)
	if !agent.IsRunning() {
		t.Error("Expected agent to be running after setting flag")
	}
}

func TestAgent_Stop(t *testing.T) {
	agent := &Agent{}
	agent.running.Store(true)

	agent.Stop()

	if agent.IsRunning() {
		t.Error("Expected agent to be stopped after Stop()")
	}
}
