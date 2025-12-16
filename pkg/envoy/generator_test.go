package envoy

import (
	"testing"
	"time"

	"github.com/vpsie/vpsie-loadbalancer/pkg/models"
)

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator("node-1", "/etc/envoy", "127.0.0.1:9901", 9901, 50000)

	if gen.nodeID != "node-1" {
		t.Errorf("nodeID = %v, want node-1", gen.nodeID)
	}
	if gen.configPath != "/etc/envoy" {
		t.Errorf("configPath = %v, want /etc/envoy", gen.configPath)
	}
	if gen.adminAddress != "127.0.0.1:9901" {
		t.Errorf("adminAddress = %v, want 127.0.0.1:9901", gen.adminAddress)
	}
	if gen.adminPort != 9901 {
		t.Errorf("adminPort = %v, want 9901", gen.adminPort)
	}
	if gen.maxConnections != 50000 {
		t.Errorf("maxConnections = %v, want 50000", gen.maxConnections)
	}
}

func TestGenerator_GenerateBootstrap(t *testing.T) {
	gen := NewGenerator("test-node", "/etc/envoy", "127.0.0.1:9901", 9901, 50000)

	data, err := gen.GenerateBootstrap()
	if err != nil {
		t.Errorf("GenerateBootstrap() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("GenerateBootstrap() returned empty data")
	}

	// Basic check that it contains expected values
	dataStr := string(data)
	if dataStr == "" {
		t.Error("Bootstrap config is empty")
	}
}

func TestGenerator_GenerateListener(t *testing.T) {
	gen := NewGenerator("test-node", "/etc/envoy", "127.0.0.1:9901", 9901, 50000)

	tests := []struct {
		name     string
		lb       *models.LoadBalancer
		wantErr  bool
	}{
		{
			name: "HTTP listener",
			lb: &models.LoadBalancer{
				ID:        "lb-1",
				Name:      "test-http",
				Protocol:  models.ProtocolHTTP,
				Algorithm: models.AlgoRoundRobin,
				Port:      80,
				Backends: []models.Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "HTTPS listener",
			lb: &models.LoadBalancer{
				ID:        "lb-2",
				Name:      "test-https",
				Protocol:  models.ProtocolHTTPS,
				Algorithm: models.AlgoLeastRequest,
				Port:      443,
				Backends: []models.Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8443, Enabled: true},
				},
				TLSConfig: &models.TLSConfig{
					CertificatePath: "/etc/certs/cert.pem",
					PrivateKeyPath:  "/etc/certs/key.pem",
					MinVersion:      "TLSv1.2",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "TCP listener",
			lb: &models.LoadBalancer{
				ID:        "lb-3",
				Name:      "test-tcp",
				Protocol:  models.ProtocolTCP,
				Algorithm: models.AlgoRandom,
				Port:      3306,
				Backends: []models.Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 3306, Enabled: true},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := gen.GenerateListener(tt.lb)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateListener() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(data) == 0 {
				t.Error("GenerateListener() returned empty data")
			}
		})
	}
}

func TestGenerator_GenerateCluster(t *testing.T) {
	gen := NewGenerator("test-node", "/etc/envoy", "127.0.0.1:9901", 9901, 50000)

	lb := &models.LoadBalancer{
		ID:        "lb-1",
		Name:      "test-lb",
		Protocol:  models.ProtocolHTTP,
		Algorithm: models.AlgoRoundRobin,
		Port:      80,
		Backends: []models.Backend{
			{ID: "be-1", Address: "10.0.0.1", Port: 8080, Weight: 100, Enabled: true},
			{ID: "be-2", Address: "10.0.0.2", Port: 8080, Weight: 50, Enabled: true},
		},
		HealthCheck: &models.HealthCheck{
			Type:               models.HealthCheckHTTP,
			Path:               "/health",
			Interval:           10,
			Timeout:            5,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := gen.GenerateCluster(lb)
	if err != nil {
		t.Errorf("GenerateCluster() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("GenerateCluster() returned empty data")
	}
}

func TestGenerator_GenerateFullConfig(t *testing.T) {
	gen := NewGenerator("test-node", "/etc/envoy", "127.0.0.1:9901", 9901, 50000)

	lb := &models.LoadBalancer{
		ID:        "lb-1",
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

	config, err := gen.GenerateFullConfig(lb)
	if err != nil {
		t.Errorf("GenerateFullConfig() error = %v", err)
	}

	if config == nil {
		t.Fatal("GenerateFullConfig() returned nil config")
	}

	if len(config.Listeners) == 0 {
		t.Error("Listeners config is empty")
	}

	if len(config.Clusters) == 0 {
		t.Error("Clusters config is empty")
	}
}
