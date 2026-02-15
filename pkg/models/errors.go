package models

import "errors"

// Load balancer validation errors
var (
	ErrInvalidID        = errors.New("invalid load balancer ID")
	ErrInvalidName      = errors.New("invalid load balancer name")
	ErrInvalidPort      = errors.New("invalid port number")
	ErrInvalidProtocol  = errors.New("invalid protocol")
	ErrNoBackends       = errors.New("no backends configured")
	ErrInvalidAlgorithm = errors.New("invalid load balancing algorithm")
	ErrMissingTLSConfig = errors.New("HTTPS protocol requires TLS configuration")
)

// Backend validation errors
var (
	ErrInvalidBackendID      = errors.New("invalid backend ID")
	ErrInvalidBackendAddress = errors.New("invalid backend address")
	ErrInvalidBackendPort    = errors.New("invalid backend port")
	ErrInvalidBackendWeight  = errors.New("invalid backend weight")
)

// Health check validation errors
var (
	ErrInvalidHealthCheckType     = errors.New("invalid health check type")
	ErrInvalidHealthCheckInterval = errors.New("invalid health check interval")
	ErrInvalidHealthCheckTimeout  = errors.New("invalid health check timeout")
	ErrHealthCheckTimeoutTooLong  = errors.New("health check timeout must be less than interval")
	ErrInvalidUnhealthyThreshold  = errors.New("invalid unhealthy threshold")
	ErrInvalidHealthyThreshold    = errors.New("invalid healthy threshold")
	ErrMissingHealthCheckPath     = errors.New("HTTP/HTTPS health check requires path")
)

// TLS configuration errors
var (
	ErrMissingCertificate = errors.New("missing certificate path")
	ErrMissingPrivateKey  = errors.New("missing private key path")
	ErrInvalidTLSVersion  = errors.New("invalid TLS version")
)
