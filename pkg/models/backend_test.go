package models

import "testing"

func TestBackend_Validate(t *testing.T) {
	tests := []struct {
		backend Backend
		wantErr error
		name    string
	}{
		{
			name: "valid backend with all fields",
			backend: Backend{
				ID:      "be-1",
				Address: "10.0.0.1",
				Port:    8080,
				Weight:  100,
				Enabled: true,
				Status:  "up",
			},
			wantErr: nil,
		},
		{
			name: "valid backend with zero weight",
			backend: Backend{
				ID:      "be-1",
				Address: "192.168.1.10",
				Port:    80,
				Weight:  0,
				Enabled: true,
			},
			wantErr: nil,
		},
		{
			name: "valid backend with hostname",
			backend: Backend{
				ID:      "be-1",
				Address: "backend.example.com",
				Port:    443,
				Weight:  50,
				Enabled: false,
			},
			wantErr: nil,
		},
		{
			name: "missing ID",
			backend: Backend{
				Address: "10.0.0.1",
				Port:    8080,
				Enabled: true,
			},
			wantErr: ErrInvalidBackendID,
		},
		{
			name: "missing address",
			backend: Backend{
				ID:      "be-1",
				Port:    8080,
				Enabled: true,
			},
			wantErr: ErrInvalidBackendAddress,
		},
		{
			name: "invalid port - zero",
			backend: Backend{
				ID:      "be-1",
				Address: "10.0.0.1",
				Port:    0,
				Enabled: true,
			},
			wantErr: ErrInvalidBackendPort,
		},
		{
			name: "invalid port - negative",
			backend: Backend{
				ID:      "be-1",
				Address: "10.0.0.1",
				Port:    -1,
				Enabled: true,
			},
			wantErr: ErrInvalidBackendPort,
		},
		{
			name: "invalid port - too high",
			backend: Backend{
				ID:      "be-1",
				Address: "10.0.0.1",
				Port:    70000,
				Enabled: true,
			},
			wantErr: ErrInvalidBackendPort,
		},
		{
			name: "invalid weight - negative",
			backend: Backend{
				ID:      "be-1",
				Address: "10.0.0.1",
				Port:    8080,
				Weight:  -1,
				Enabled: true,
			},
			wantErr: ErrInvalidBackendWeight,
		},
		{
			name: "edge case - port 1",
			backend: Backend{
				ID:      "be-1",
				Address: "10.0.0.1",
				Port:    1,
				Enabled: true,
			},
			wantErr: nil,
		},
		{
			name: "edge case - port 65535",
			backend: Backend{
				ID:      "be-1",
				Address: "10.0.0.1",
				Port:    65535,
				Enabled: true,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backend.Validate()
			if err != tt.wantErr {
				t.Errorf("Backend.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBackend_IsHealthy(t *testing.T) {
	tests := []struct {
		backend  Backend
		name     string
		expected bool
	}{
		{
			name: "healthy backend - enabled and up",
			backend: Backend{
				Enabled: true,
				Status:  "up",
			},
			expected: true,
		},
		{
			name: "unhealthy - disabled",
			backend: Backend{
				Enabled: false,
				Status:  "up",
			},
			expected: false,
		},
		{
			name: "unhealthy - down",
			backend: Backend{
				Enabled: true,
				Status:  "down",
			},
			expected: false,
		},
		{
			name: "unhealthy - unknown status",
			backend: Backend{
				Enabled: true,
				Status:  "unknown",
			},
			expected: false,
		},
		{
			name: "unhealthy - disabled and down",
			backend: Backend{
				Enabled: false,
				Status:  "down",
			},
			expected: false,
		},
		{
			name: "unhealthy - empty status",
			backend: Backend{
				Enabled: true,
				Status:  "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.backend.IsHealthy()
			if result != tt.expected {
				t.Errorf("Backend.IsHealthy() = %v, want %v", result, tt.expected)
			}
		})
	}
}
