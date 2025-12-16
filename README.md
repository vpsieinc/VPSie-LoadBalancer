# VPSie Load Balancer

A cloud-native load balancer solution for VPSie cloud platform, powered by Envoy Proxy.

## Features

- **HTTP/HTTPS Load Balancing** with TLS/SSL termination
- **TCP Load Balancing** for Layer 4 traffic
- **Multi-Architecture Support**: amd64 and arm64
- **Built on Envoy Proxy** for modern, performant load balancing
- **VPSie API Integration** for seamless configuration management
- **Active Health Checking** for backend servers
- **Prometheus Metrics** for comprehensive observability
- **Hot Configuration Reload** with zero downtime

## Architecture

The load balancer consists of two main components:

1. **Envoy Proxy** - High-performance data plane for traffic routing
2. **Control Plane Agent** - Go-based agent that:
   - Polls VPSie API for configuration updates
   - Generates Envoy configuration from VPSie specs
   - Manages TLS certificates
   - Performs hot reloads
   - Exports metrics

## Quick Start

### Building Images

```bash
# Build both architectures
make build-all

# Build specific architecture
make build-amd64
make build-arm64
```

### Deploying to Proxmox

```bash
# Import the qcow2 image
qm create 100 --name vpsie-lb-01 --memory 4096 --cores 4 --net0 virtio,bridge=vmbr0
qm importdisk 100 output/vpsie-lb-debian-12-amd64-1.0.0.qcow2 local-lvm
qm set 100 --scsihw virtio-scsi-pci --scsi0 local-lvm:vm-100-disk-0
qm set 100 --boot c --bootdisk scsi0
qm start 100
```

### Configuration

The agent configuration is located at `/etc/vpsie-lb/agent.yaml`:

```yaml
vpsie:
  api_url: https://api.vpsie.com/v1
  api_key_file: /etc/vpsie-lb/api-key
  loadbalancer_id: lb-123456
  poll_interval: 30s

envoy:
  config_path: /etc/envoy/dynamic
  admin_address: 127.0.0.1:9901

logging:
  level: info
  format: json
```

## Project Structure

```
.
├── cmd/agent/              # Main agent binary
├── pkg/
│   ├── agent/             # Agent core logic
│   ├── envoy/             # Envoy configuration generation
│   ├── models/            # Data structures
│   └── utils/             # Utilities
├── configs/               # Default configurations
├── packer/                # Image build templates
├── systemd/               # Service files
└── docs/                  # Documentation
```

## Development

### Prerequisites

- Go 1.21+
- Packer 1.9+
- QEMU
- Make

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
make test-integration
```

### Building Locally

```bash
# Build agent binary
make build-agent

# Build for specific architecture
GOARCH=arm64 make build-agent
```

## Documentation

- [Architecture](docs/architecture.md)
- [Deployment Guide](docs/deployment.md)
- [Configuration Reference](docs/configuration.md)
- [VPSie API Integration](docs/vpsie-integration.md)

## Monitoring

Metrics are exposed on the Envoy admin interface:

```bash
# Prometheus metrics
curl http://127.0.0.1:9901/stats/prometheus

# Health check
curl http://127.0.0.1:9901/ready
```

## License

Copyright (c) 2025 VPSie
