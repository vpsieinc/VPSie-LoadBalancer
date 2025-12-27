# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VPSie Load Balancer is a cloud-native load balancer solution built on Envoy Proxy. It consists of a Go-based control plane agent that manages Envoy configuration by polling the VPSie API for load balancer specifications and performing hot configuration reloads.

## Architecture

### Two-Component Design

1. **Envoy Proxy** - High-performance data plane handling actual traffic routing
2. **Control Plane Agent** (Go) - Manages Envoy lifecycle:
   - Polls VPSie API for configuration updates (default: 30s interval)
   - Generates Envoy configuration from VPSie API specifications
   - Manages TLS certificates
   - Performs hot reloads using Envoy's epoch-based restart mechanism
   - Exports Prometheus metrics via Envoy admin interface

### Configuration Flow

The agent (pkg/agent/agent.go:78) implements a reconciliation loop:
1. Fetches LoadBalancer config from VPSie API
2. Computes SHA-256 hash of configuration for change detection
3. On change: generates Envoy config, validates it, backs up current config
4. Performs hot reload using Envoy's restart mechanism with epoch tracking
5. Rolls back on failure

### Key Packages

- `pkg/agent/` - Main control plane logic, VPSie API client, configuration loading
- `pkg/envoy/` - Envoy configuration generation from Go templates, validation, hot reload management
- `pkg/models/` - Data structures (LoadBalancer, Backend, HealthCheck, TLSConfig)
- `cmd/agent/` - Main entry point with signal handling and graceful shutdown

## Common Commands

### Development

```bash
# Run all tests with race detection and coverage
make test

# Run only unit tests
make test-unit

# Run integration tests
make test-integration

# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet
```

### Building

```bash
# Build agent binary for current platform
make build-agent

# Build for specific architecture
GOARCH=arm64 make build-agent
GOARCH=amd64 make build-agent

# Build agent for all architectures (amd64 and arm64)
make build-agent-all

# Build VM images using Packer
make build-amd64    # Build amd64 qcow2 image
make build-arm64    # Build arm64 qcow2 image
make build-images   # Build both architectures
```

### Dependencies

```bash
# Download and tidy dependencies
make deps

# Install development tools (golangci-lint)
make install-tools
```

## Important Implementation Details

### Configuration Validation

All configuration from the VPSie API undergoes strict validation (models/loadbalancer.go:56):
- IDs and Names must match safe identifier regex `^[a-zA-Z0-9_-]+$` to prevent template injection
- Maximum lengths enforced (ID: 64 chars, Name: 255 chars)
- Backend IP addresses validated against private/localhost ranges for security
- TLS configuration validated when protocol is HTTPS

### Envoy Hot Reload

The reloader (pkg/envoy/reloader.go) uses Envoy's hot restart mechanism:
- Tracks epoch number for each reload
- New Envoy process starts with incremented epoch
- Drains connections from old process
- On failure, restores backed-up configuration and notifies VPSie API

### API Client Security

The VPSie client (pkg/agent/vpsie_client.go) includes:
- SSRF protection: blocks requests to private IPs, localhost, and metadata endpoints
- Response size limiting (10MB max) to prevent DoS
- URL validation requiring HTTPS for production
- Automatic HTTP client timeout configuration

### Error Handling

Critical failures trigger VPSie API notifications (agent.go:166):
- Config reload failures with restore attempts
- System sends events to VPSie API including error details and epoch information
- Backup/restore mechanism prevents inconsistent states

## Configuration File

Agent configuration at `/etc/vpsie-lb/agent.yaml`:

```yaml
vpsie:
  api_url: https://api.vpsie.com/v1
  api_key_file: /etc/vpsie-lb/api-key
  loadbalancer_id: lb-123456
  poll_interval: 30s

envoy:
  config_path: /etc/envoy/dynamic
  binary_path: /usr/bin/envoy
  admin_address: 127.0.0.1:9901
  admin_port: 9901
  pid_file: /var/run/envoy.pid
  max_connections: 50000

logging:
  level: info
  format: json
```

## Testing Patterns

- Unit tests use table-driven patterns
- Mocks avoid external dependencies (no actual API calls)
- Integration tests validate end-to-end flows
- Test files follow `*_test.go` naming convention

## Deployment

The project uses Packer to build qcow2 VM images for Proxmox deployment. Build artifacts go to `output/` directory. The agent runs as a systemd service (see `systemd/` directory).
