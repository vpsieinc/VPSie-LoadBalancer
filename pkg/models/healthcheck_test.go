package models

import "testing"

func TestHealthCheck_Validate(t *testing.T) {
	tests := []struct {
		hc      HealthCheck
		wantErr error
		name    string
	}{
		{
			name: "valid TCP health check",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           10,
				Timeout:            5,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
			wantErr: nil,
		},
		{
			name: "valid HTTP health check",
			hc: HealthCheck{
				Type:               HealthCheckHTTP,
				Path:               "/health",
				Interval:           15,
				Timeout:            5,
				HealthyThreshold:   3,
				UnhealthyThreshold: 2,
				ExpectedStatus:     []int{200, 204},
			},
			wantErr: nil,
		},
		{
			name: "valid HTTPS health check with headers",
			hc: HealthCheck{
				Type:               HealthCheckHTTPS,
				Path:               "/api/health",
				Headers:            map[string]string{"Authorization": "Bearer token"},
				Interval:           20,
				Timeout:            10,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
				ExpectedStatus:     []int{200},
			},
			wantErr: nil,
		},
		{
			name: "invalid type",
			hc: HealthCheck{
				Type:               HealthCheckType("invalid"),
				Interval:           10,
				Timeout:            5,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
			wantErr: ErrInvalidHealthCheckType,
		},
		{
			name: "invalid interval - zero",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           0,
				Timeout:            5,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
			wantErr: ErrInvalidHealthCheckInterval,
		},
		{
			name: "invalid interval - negative",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           -10,
				Timeout:            5,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
			wantErr: ErrInvalidHealthCheckInterval,
		},
		{
			name: "invalid timeout - zero",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           10,
				Timeout:            0,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
			wantErr: ErrInvalidHealthCheckTimeout,
		},
		{
			name: "invalid timeout - negative",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           10,
				Timeout:            -5,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
			wantErr: ErrInvalidHealthCheckTimeout,
		},
		{
			name: "timeout equals interval",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           10,
				Timeout:            10,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
			wantErr: ErrHealthCheckTimeoutTooLong,
		},
		{
			name: "timeout greater than interval",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           10,
				Timeout:            15,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
			wantErr: ErrHealthCheckTimeoutTooLong,
		},
		{
			name: "invalid unhealthy threshold - zero",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           10,
				Timeout:            5,
				HealthyThreshold:   2,
				UnhealthyThreshold: 0,
			},
			wantErr: ErrInvalidUnhealthyThreshold,
		},
		{
			name: "invalid unhealthy threshold - negative",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           10,
				Timeout:            5,
				HealthyThreshold:   2,
				UnhealthyThreshold: -1,
			},
			wantErr: ErrInvalidUnhealthyThreshold,
		},
		{
			name: "invalid healthy threshold - zero",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           10,
				Timeout:            5,
				HealthyThreshold:   0,
				UnhealthyThreshold: 3,
			},
			wantErr: ErrInvalidHealthyThreshold,
		},
		{
			name: "invalid healthy threshold - negative",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           10,
				Timeout:            5,
				HealthyThreshold:   -2,
				UnhealthyThreshold: 3,
			},
			wantErr: ErrInvalidHealthyThreshold,
		},
		{
			name: "HTTP health check missing path",
			hc: HealthCheck{
				Type:               HealthCheckHTTP,
				Path:               "",
				Interval:           10,
				Timeout:            5,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
			wantErr: ErrMissingHealthCheckPath,
		},
		{
			name: "HTTPS health check missing path",
			hc: HealthCheck{
				Type:               HealthCheckHTTPS,
				Interval:           10,
				Timeout:            5,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
			wantErr: ErrMissingHealthCheckPath,
		},
		{
			name: "TCP health check does not require path",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           10,
				Timeout:            5,
				HealthyThreshold:   2,
				UnhealthyThreshold: 3,
			},
			wantErr: nil,
		},
		{
			name: "edge case - timeout just less than interval",
			hc: HealthCheck{
				Type:               HealthCheckTCP,
				Interval:           10,
				Timeout:            9,
				HealthyThreshold:   1,
				UnhealthyThreshold: 1,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.hc.Validate()
			if err != tt.wantErr {
				t.Errorf("HealthCheck.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHealthCheck_IsHTTPBased(t *testing.T) {
	tests := []struct {
		hc       HealthCheck
		name     string
		expected bool
	}{
		{
			name:     "HTTP health check",
			hc:       HealthCheck{Type: HealthCheckHTTP},
			expected: true,
		},
		{
			name:     "HTTPS health check",
			hc:       HealthCheck{Type: HealthCheckHTTPS},
			expected: true,
		},
		{
			name:     "TCP health check",
			hc:       HealthCheck{Type: HealthCheckTCP},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.hc.IsHTTPBased()
			if result != tt.expected {
				t.Errorf("HealthCheck.IsHTTPBased() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHealthCheckTypeConstants(t *testing.T) {
	tests := []struct {
		hcType   HealthCheckType
		expected string
	}{
		{HealthCheckTCP, "tcp"},
		{HealthCheckHTTP, "http"},
		{HealthCheckHTTPS, "https"},
	}

	for _, tt := range tests {
		t.Run(string(tt.hcType), func(t *testing.T) {
			if string(tt.hcType) != tt.expected {
				t.Errorf("HealthCheckType constant = %v, want %v", tt.hcType, tt.expected)
			}
		})
	}
}
