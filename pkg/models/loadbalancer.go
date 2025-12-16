package models

import "time"

// Protocol defines the load balancer protocol type
type Protocol string

const (
	ProtocolHTTP  Protocol = "http"
	ProtocolHTTPS Protocol = "https"
	ProtocolTCP   Protocol = "tcp"
)

// LoadBalancingAlgo defines the load balancing algorithm
type LoadBalancingAlgo string

const (
	AlgoRoundRobin   LoadBalancingAlgo = "round_robin"
	AlgoLeastRequest LoadBalancingAlgo = "least_request"
	AlgoRandom       LoadBalancingAlgo = "random"
	AlgoRingHash     LoadBalancingAlgo = "ring_hash"
)

// LoadBalancer represents the main load balancer configuration
type LoadBalancer struct {
	ID             string            `json:"id" yaml:"id"`
	Name           string            `json:"name" yaml:"name"`
	Protocol       Protocol          `json:"protocol" yaml:"protocol"`
	Port           int               `json:"port" yaml:"port"`
	Algorithm      LoadBalancingAlgo `json:"algorithm" yaml:"algorithm"`
	Backends       []Backend         `json:"backends" yaml:"backends"`
	HealthCheck    *HealthCheck      `json:"health_check,omitempty" yaml:"health_check,omitempty"`
	TLSConfig      *TLSConfig        `json:"tls_config,omitempty" yaml:"tls_config,omitempty"`
	Timeouts       *Timeouts         `json:"timeouts,omitempty" yaml:"timeouts,omitempty"`
	MaxConnections int               `json:"max_connections,omitempty" yaml:"max_connections,omitempty"`
	CreatedAt      time.Time         `json:"created_at" yaml:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" yaml:"updated_at"`
}

// Timeouts defines timeout configuration for the load balancer
type Timeouts struct {
	Connect int `json:"connect" yaml:"connect"` // seconds
	Idle    int `json:"idle" yaml:"idle"`       // seconds
	Request int `json:"request" yaml:"request"` // seconds
}

// Validate validates the load balancer configuration
func (lb *LoadBalancer) Validate() error {
	if err := lb.validateBasicFields(); err != nil {
		return err
	}
	if err := lb.validateBackends(); err != nil {
		return err
	}
	if err := lb.validateTLSConfig(); err != nil {
		return err
	}
	if err := lb.validateHealthCheck(); err != nil {
		return err
	}
	return nil
}

func (lb *LoadBalancer) validateBasicFields() error {
	if lb.ID == "" {
		return ErrInvalidID
	}
	if lb.Name == "" {
		return ErrInvalidName
	}
	if lb.Port <= 0 || lb.Port > 65535 {
		return ErrInvalidPort
	}
	if lb.Protocol != ProtocolHTTP && lb.Protocol != ProtocolHTTPS && lb.Protocol != ProtocolTCP {
		return ErrInvalidProtocol
	}
	return nil
}

func (lb *LoadBalancer) validateBackends() error {
	if len(lb.Backends) == 0 {
		return ErrNoBackends
	}
	for _, backend := range lb.Backends {
		if err := backend.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (lb *LoadBalancer) validateTLSConfig() error {
	if lb.Protocol == ProtocolHTTPS && lb.TLSConfig == nil {
		return ErrMissingTLSConfig
	}
	if lb.TLSConfig != nil {
		if err := lb.TLSConfig.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (lb *LoadBalancer) validateHealthCheck() error {
	if lb.HealthCheck != nil {
		if err := lb.HealthCheck.Validate(); err != nil {
			return err
		}
	}
	return nil
}
