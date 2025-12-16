package models

import (
	"testing"
	"time"
)

func TestLoadBalancer_Validate(t *testing.T) {
	tests := []struct {
		name    string
		wantErr error
		lb      LoadBalancer
	}{
		{
			name: "valid HTTP load balancer",
			lb: LoadBalancer{
				ID:        "lb-123",
				Name:      "test-lb",
				Protocol:  ProtocolHTTP,
				Algorithm: AlgoRoundRobin,
				Port:      80,
				Backends: []Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: nil,
		},
		{
			name: "valid HTTPS load balancer with TLS",
			lb: LoadBalancer{
				ID:        "lb-123",
				Name:      "test-lb",
				Protocol:  ProtocolHTTPS,
				Algorithm: AlgoLeastRequest,
				Port:      443,
				Backends: []Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
				TLSConfig: &TLSConfig{
					CertificatePath: "/etc/certs/cert.pem",
					PrivateKeyPath:  "/etc/certs/key.pem",
					MinVersion:      "TLSv1.2",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: nil,
		},
		{
			name: "valid TCP load balancer",
			lb: LoadBalancer{
				ID:        "lb-123",
				Name:      "test-lb",
				Protocol:  ProtocolTCP,
				Algorithm: AlgoRandom,
				Port:      3306,
				Backends: []Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 3306, Enabled: true},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: nil,
		},
		{
			name: "missing ID",
			lb: LoadBalancer{
				Name:     "test-lb",
				Protocol: ProtocolHTTP,
				Port:     80,
				Backends: []Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
			},
			wantErr: ErrInvalidID,
		},
		{
			name: "missing name",
			lb: LoadBalancer{
				ID:       "lb-123",
				Protocol: ProtocolHTTP,
				Port:     80,
				Backends: []Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
			},
			wantErr: ErrInvalidName,
		},
		{
			name: "invalid port - zero",
			lb: LoadBalancer{
				ID:       "lb-123",
				Name:     "test-lb",
				Protocol: ProtocolHTTP,
				Port:     0,
				Backends: []Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
			},
			wantErr: ErrInvalidPort,
		},
		{
			name: "invalid port - too high",
			lb: LoadBalancer{
				ID:       "lb-123",
				Name:     "test-lb",
				Protocol: ProtocolHTTP,
				Port:     70000,
				Backends: []Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
			},
			wantErr: ErrInvalidPort,
		},
		{
			name: "invalid protocol",
			lb: LoadBalancer{
				ID:       "lb-123",
				Name:     "test-lb",
				Protocol: Protocol("invalid"),
				Port:     80,
				Backends: []Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
			},
			wantErr: ErrInvalidProtocol,
		},
		{
			name: "no backends",
			lb: LoadBalancer{
				ID:       "lb-123",
				Name:     "test-lb",
				Protocol: ProtocolHTTP,
				Port:     80,
				Backends: []Backend{},
			},
			wantErr: ErrNoBackends,
		},
		{
			name: "HTTPS without TLS config",
			lb: LoadBalancer{
				ID:       "lb-123",
				Name:     "test-lb",
				Protocol: ProtocolHTTPS,
				Port:     443,
				Backends: []Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
			},
			wantErr: ErrMissingTLSConfig,
		},
		{
			name: "invalid backend",
			lb: LoadBalancer{
				ID:       "lb-123",
				Name:     "test-lb",
				Protocol: ProtocolHTTP,
				Port:     80,
				Backends: []Backend{
					{ID: "", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
			},
			wantErr: ErrInvalidBackendID,
		},
		{
			name: "valid with health check",
			lb: LoadBalancer{
				ID:        "lb-123",
				Name:      "test-lb",
				Protocol:  ProtocolHTTP,
				Algorithm: AlgoRoundRobin,
				Port:      80,
				Backends: []Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
				HealthCheck: &HealthCheck{
					Type:               HealthCheckHTTP,
					Path:               "/health",
					Interval:           10,
					Timeout:            5,
					HealthyThreshold:   2,
					UnhealthyThreshold: 3,
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: nil,
		},
		{
			name: "invalid health check",
			lb: LoadBalancer{
				ID:       "lb-123",
				Name:     "test-lb",
				Protocol: ProtocolHTTP,
				Port:     80,
				Backends: []Backend{
					{ID: "be-1", Address: "10.0.0.1", Port: 8080, Enabled: true},
				},
				HealthCheck: &HealthCheck{
					Type:               HealthCheckHTTP,
					Path:               "", // Missing path for HTTP health check
					Interval:           10,
					Timeout:            5,
					HealthyThreshold:   2,
					UnhealthyThreshold: 3,
				},
			},
			wantErr: ErrMissingHealthCheckPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lb.Validate()
			if err != tt.wantErr {
				t.Errorf("LoadBalancer.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProtocolConstants(t *testing.T) {
	tests := []struct {
		protocol Protocol
		expected string
	}{
		{ProtocolHTTP, "http"},
		{ProtocolHTTPS, "https"},
		{ProtocolTCP, "tcp"},
	}

	for _, tt := range tests {
		t.Run(string(tt.protocol), func(t *testing.T) {
			if string(tt.protocol) != tt.expected {
				t.Errorf("Protocol constant = %v, want %v", tt.protocol, tt.expected)
			}
		})
	}
}

func TestLoadBalancingAlgoConstants(t *testing.T) {
	tests := []struct {
		algo     LoadBalancingAlgo
		expected string
	}{
		{AlgoRoundRobin, "round_robin"},
		{AlgoLeastRequest, "least_request"},
		{AlgoRandom, "random"},
		{AlgoRingHash, "ring_hash"},
	}

	for _, tt := range tests {
		t.Run(string(tt.algo), func(t *testing.T) {
			if string(tt.algo) != tt.expected {
				t.Errorf("LoadBalancingAlgo constant = %v, want %v", tt.algo, tt.expected)
			}
		})
	}
}
