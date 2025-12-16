# VPSie Load Balancer Architecture

## Overview

The VPSie Load Balancer is a cloud-native load balancing solution built on Envoy Proxy, designed to run on virtual machines in Proxmox infrastructure. It provides HTTP/HTTPS and TCP load balancing with seamless VPSie API integration.

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    VPSie API (External)                      │
│              (Configuration & Orchestration)                 │
└────────────────────────┬────────────────────────────────────┘
                         │ Poll every 30s
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  Load Balancer VM (Proxmox)                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Control Plane (Management API & Config Agent)      │   │
│  │  - VPSie API Client                                 │   │
│  │  - Configuration Manager                            │   │
│  │  - Config Generator                                 │   │
│  └─────────────────┬───────────────────────────────────┘   │
│                    │                                         │
│  ┌─────────────────┴───────────────────────────────────┐   │
│  │  Data Plane (Traffic Processing)                    │   │
│  │  ┌──────────────────────────────────────────┐       │   │
│  │  │ Envoy Proxy                              │       │   │
│  │  │ - L4/L7 Load Balancing                   │       │   │
│  │  │ - TLS/SSL Termination                    │       │   │
│  │  │ - Health Checking                        │       │   │
│  │  └──────────────────────────────────────────┘       │   │
│  └─────────────────────────────────────────────────────┘   │
│                    │                                         │
│  ┌─────────────────┴───────────────────────────────────┐   │
│  │  Observability                                       │   │
│  │  - Prometheus Metrics (Envoy Admin API)             │   │
│  │  - Structured Logging (systemd journald)            │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │  Backend Pool        │
              │  (Customer VMs)      │
              └──────────────────────┘
```

## Components

### Control Plane Agent

The control plane agent is written in Go and provides:

- **VPSie API Integration**: Polls VPSie API for configuration changes
- **Config Generation**: Transforms VPSie config into Envoy YAML
- **Config Validation**: Validates generated configs before applying
- **Hot Reload**: Manages Envoy configuration updates
- **Metrics Collection**: Exports agent health metrics

**Key Files:**
- `pkg/agent/agent.go` - Main agent logic
- `pkg/agent/vpsie_client.go` - VPSie API client
- `pkg/agent/config.go` - Agent configuration

### Data Plane (Envoy Proxy)

Envoy Proxy handles all traffic processing:

- **Listeners**: HTTP (80), HTTPS (443), TCP (custom ports)
- **Clusters**: Backend server groups with health checking
- **Load Balancing**: Round robin, least request, random, ring hash
- **Health Checks**: TCP, HTTP, HTTPS
- **TLS Termination**: Modern TLS 1.2/1.3 with configurable ciphers
- **Circuit Breaking**: Automatic failure detection and recovery

**Config Files:**
- `/etc/envoy/bootstrap.yaml` - Static bootstrap config
- `/etc/envoy/dynamic/listeners.yaml` - Generated listeners
- `/etc/envoy/dynamic/clusters.yaml` - Generated clusters

### Configuration Management

Configuration flow:

1. **Fetch**: Agent polls VPSie API for latest config
2. **Validate**: Config is validated against data models
3. **Generate**: Envoy YAML is generated from VPSie config
4. **Backup**: Current config is backed up
5. **Apply**: New config is written atomically
6. **Reload**: Envoy detects changes and reloads

**Reconciliation Loop:**
- Runs every 30 seconds (configurable)
- Compares desired state (VPSie) vs actual state (local)
- Only applies changes when configuration differs
- Includes rollback on failure

## Data Models

### LoadBalancer

```go
type LoadBalancer struct {
    ID              string
    Name            string
    Protocol        Protocol  // http, https, tcp
    Port            int
    Algorithm       LoadBalancingAlgo
    Backends        []Backend
    HealthCheck     *HealthCheck
    TLSConfig       *TLSConfig
    Timeouts        *Timeouts
}
```

### Backend

```go
type Backend struct {
    ID      string
    Address string
    Port    int
    Weight  int
    Enabled bool
}
```

### HealthCheck

```go
type HealthCheck struct {
    Type               HealthCheckType  // tcp, http, https
    Interval           int
    Timeout            int
    UnhealthyThreshold int
    HealthyThreshold   int
    Path               string
    ExpectedStatus     []int
}
```

## Security

### TLS/SSL Configuration

- **Supported Versions**: TLS 1.2, TLS 1.3
- **Cipher Suites**: Modern, secure ciphers only
- **ALPN**: HTTP/2 and HTTP/1.1
- **Certificate Management**: File-based, hot reloadable

### System Security

- **Least Privilege**: Envoy runs as dedicated `envoy` user
- **File Permissions**: Config files are protected (mode 600/644)
- **API Key Storage**: Stored separately, not in config files
- **Resource Limits**: ulimits and systemd limits enforced

## Monitoring

### Metrics

Envoy exposes Prometheus metrics on `127.0.0.1:9901/stats/prometheus`:

- **Listener Metrics**: Connections, bytes, requests
- **Cluster Metrics**: Health, requests, response times
- **System Metrics**: Memory, CPU, file descriptors

### Logging

- **Agent Logs**: JSON format to systemd journal
- **Envoy Logs**: Structured access and error logs
- **Log Rotation**: Automatic via systemd

### Health Checks

- **Envoy Admin**: `curl http://127.0.0.1:9901/ready`
- **Agent Status**: `systemctl status vpsie-lb-agent`
- **Backend Health**: Monitored by Envoy, reported to VPSie

## Scalability

### Performance Tuning

- **Kernel Tuning**: sysctl optimizations for network performance
- **Connection Limits**: 50,000 concurrent connections
- **File Descriptors**: 1M+ for Envoy process
- **Circuit Breakers**: Prevent cascade failures

### Resource Requirements

**Minimum:**
- CPU: 2 cores
- Memory: 2GB RAM
- Disk: 10GB
- Network: 1 Gbps

**Recommended:**
- CPU: 4 cores
- Memory: 4GB RAM
- Disk: 20GB (with logging)
- Network: 10 Gbps

## Failure Handling

### Agent Failures

- **Systemd Restart**: Auto-restart on crash
- **API Failures**: Continue with cached config
- **Config Errors**: Rollback to previous working config

### Envoy Failures

- **Health Checks**: Automatic backend failure detection
- **Circuit Breaking**: Prevent overload
- **Graceful Shutdown**: Connection draining

## Future Enhancements

1. **xDS API**: Move from static config to dynamic xDS
2. **High Availability**: Active-active or active-passive pairs
3. **Auto-scaling**: Dynamic backend pool management
4. **Advanced Routing**: Header-based, weighted routing
5. **Rate Limiting**: Per-client rate limits
6. **WAF Integration**: Web application firewall
