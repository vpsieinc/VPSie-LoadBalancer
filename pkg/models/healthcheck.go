package models

// HealthCheckType defines the type of health check
type HealthCheckType string

const (
	HealthCheckTCP   HealthCheckType = "tcp"
	HealthCheckHTTP  HealthCheckType = "http"
	HealthCheckHTTPS HealthCheckType = "https"
)

// HealthCheck represents health check configuration
type HealthCheck struct {
	ExpectedStatus     []int             `json:"expected_status,omitempty" yaml:"expected_status,omitempty"`
	Path               string            `json:"path,omitempty" yaml:"path,omitempty"` // for HTTP/HTTPS
	Type               HealthCheckType   `json:"type" yaml:"type"`
	Headers            map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Interval           int               `json:"interval" yaml:"interval"`                       // seconds
	Timeout            int               `json:"timeout" yaml:"timeout"`                         // seconds
	UnhealthyThreshold int               `json:"unhealthy_threshold" yaml:"unhealthy_threshold"` // consecutive failures
	HealthyThreshold   int               `json:"healthy_threshold" yaml:"healthy_threshold"`     // consecutive successes
}

// Validate validates the health check configuration
func (h *HealthCheck) Validate() error {
	if h.Type != HealthCheckTCP && h.Type != HealthCheckHTTP && h.Type != HealthCheckHTTPS {
		return ErrInvalidHealthCheckType
	}
	if h.Interval <= 0 {
		return ErrInvalidHealthCheckInterval
	}
	if h.Timeout <= 0 {
		return ErrInvalidHealthCheckTimeout
	}
	if h.Timeout >= h.Interval {
		return ErrHealthCheckTimeoutTooLong
	}
	if h.UnhealthyThreshold <= 0 {
		return ErrInvalidUnhealthyThreshold
	}
	if h.HealthyThreshold <= 0 {
		return ErrInvalidHealthyThreshold
	}

	// HTTP/HTTPS health checks require a path
	if (h.Type == HealthCheckHTTP || h.Type == HealthCheckHTTPS) && h.Path == "" {
		return ErrMissingHealthCheckPath
	}

	return nil
}

// IsHTTPBased returns true if the health check is HTTP or HTTPS
func (h *HealthCheck) IsHTTPBased() bool {
	return h.Type == HealthCheckHTTP || h.Type == HealthCheckHTTPS
}
