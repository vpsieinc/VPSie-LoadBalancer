# VPSie Load Balancer Configuration Guide

## Agent Configuration

### Configuration File: `/etc/vpsie-lb/agent.yaml`

```yaml
vpsie:
  # VPSie API endpoint
  api_url: https://api.vpsie.com/v1

  # Path to file containing API key
  api_key_file: /etc/vpsie-lb/api-key

  # Load balancer ID from VPSie
  loadbalancer_id: lb-your-id-here

  # How often to poll VPSie API for config changes
  poll_interval: 30s

envoy:
  # Directory for dynamic Envoy configs
  config_path: /etc/envoy/dynamic

  # Envoy admin interface address
  admin_address: 127.0.0.1:9901

  # Path to Envoy binary
  binary_path: /usr/bin/envoy

logging:
  # Log level: debug, info, warn, error
  level: info

  # Log format: json, text
  format: json
```

### Environment Variables

```bash
# Alternative to config file
export VPSIE_API_KEY="your-api-key"
export VPSIE_API_URL="https://api.vpsie.com/v1"
export VPSIE_LB_ID="lb-your-id"
```

## VPSie API Configuration

### Load Balancer Configuration

The agent fetches configuration from VPSie API. Example response:

```json
{
  "id": "lb-123456",
  "name": "production-lb",
  "protocol": "https",
  "port": 443,
  "algorithm": "least_request",
  "backends": [
    {
      "id": "backend-1",
      "address": "10.0.1.10",
      "port": 8080,
      "weight": 100,
      "enabled": true
    },
    {
      "id": "backend-2",
      "address": "10.0.1.11",
      "port": 8080,
      "weight": 100,
      "enabled": true
    }
  ],
  "health_check": {
    "type": "http",
    "path": "/health",
    "interval": 10,
    "timeout": 5,
    "healthy_threshold": 2,
    "unhealthy_threshold": 3,
    "expected_status": [200, 204]
  },
  "tls_config": {
    "certificate_path": "/etc/vpsie-lb/certs/cert.pem",
    "private_key_path": "/etc/vpsie-lb/certs/key.pem",
    "min_version": "TLSv1.2",
    "alpn": ["h2", "http/1.1"]
  },
  "timeouts": {
    "connect": 5,
    "idle": 300,
    "request": 60
  },
  "max_connections": 10000
}
```

### Supported Protocols

- **HTTP**: Plain HTTP traffic on any port
- **HTTPS**: TLS-terminated HTTPS traffic
- **TCP**: Layer 4 TCP proxy (any protocol)

### Load Balancing Algorithms

- **round_robin**: Distribute requests evenly across backends
- **least_request**: Send to backend with fewest active requests
- **random**: Random selection
- **ring_hash**: Consistent hashing (for session persistence)

### Health Check Types

#### TCP Health Check

```json
{
  "type": "tcp",
  "interval": 10,
  "timeout": 5,
  "healthy_threshold": 2,
  "unhealthy_threshold": 3
}
```

#### HTTP Health Check

```json
{
  "type": "http",
  "path": "/health",
  "interval": 10,
  "timeout": 5,
  "healthy_threshold": 2,
  "unhealthy_threshold": 3,
  "expected_status": [200, 204],
  "headers": {
    "Host": "example.com"
  }
}
```

## Envoy Configuration

### Bootstrap Configuration: `/etc/envoy/bootstrap.yaml`

```yaml
node:
  id: vpsie-lb-node-01
  cluster: vpsie-loadbalancers

static_resources:
  listeners: []
  clusters: []

dynamic_resources:
  lds_config:
    path: /etc/envoy/dynamic/listeners.yaml
  cds_config:
    path: /etc/envoy/dynamic/clusters.yaml

admin:
  address:
    socket_address:
      address: 127.0.0.1
      port_value: 9901
  access_log:
    - name: envoy.access_loggers.file
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog
        path: /var/log/envoy/admin.log

layered_runtime:
  layers:
    - name: static_layer
      static_layer:
        overload:
          global_downstream_max_connections: 50000
```

### Dynamic Configurations

Generated automatically by the agent:

- `/etc/envoy/dynamic/listeners.yaml` - Listeners configuration
- `/etc/envoy/dynamic/clusters.yaml` - Clusters configuration

**Do not edit these files manually!** They are generated from VPSie API.

## TLS/SSL Configuration

### Certificate Files

Place certificates in `/etc/vpsie-lb/certs/`:

```
/etc/vpsie-lb/certs/
├── example.com/
│   ├── cert.pem      # Certificate chain
│   ├── key.pem       # Private key
│   └── ca.pem        # CA certificate (optional)
```

### Permissions

```bash
chown -R root:root /etc/vpsie-lb/certs/
chmod 700 /etc/vpsie-lb/certs/
chmod 600 /etc/vpsie-lb/certs/*/key.pem
chmod 644 /etc/vpsie-lb/certs/*/cert.pem
```

### Supported TLS Versions

- TLSv1.2 (default minimum)
- TLSv1.3 (recommended)

### Cipher Suites

Default secure ciphers:
- ECDHE-ECDSA-AES128-GCM-SHA256
- ECDHE-RSA-AES128-GCM-SHA256
- ECDHE-ECDSA-AES256-GCM-SHA384
- ECDHE-RSA-AES256-GCM-SHA384
- ECDHE-ECDSA-CHACHA20-POLY1305
- ECDHE-RSA-CHACHA20-POLY1305

## System Tuning

### Kernel Parameters: `/etc/sysctl.d/99-vpsie-lb.conf`

```bash
# Network performance
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 8192
net.ipv4.ip_local_port_range = 1024 65535
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 30
net.core.netdev_max_backlog = 5000

# Connection tracking
net.netfilter.nf_conntrack_max = 262144
net.netfilter.nf_conntrack_tcp_timeout_established = 432000

# Memory
vm.swappiness = 10
```

Apply with:

```bash
sysctl -p /etc/sysctl.d/99-vpsie-lb.conf
```

### File Limits: `/etc/security/limits.d/99-vpsie-lb.conf`

```bash
* soft nofile 65536
* hard nofile 65536
* soft nproc 4096
* hard nproc 4096
```

### Systemd Service Limits

In `/etc/systemd/system/envoy.service`:

```ini
[Service]
LimitNOFILE=1048576
LimitNPROC=512
```

## Monitoring Configuration

### Prometheus Metrics

Available at: `http://127.0.0.1:9901/stats/prometheus`

Key metrics:
- `envoy_listener_downstream_cx_total` - Total connections
- `envoy_cluster_membership_healthy` - Healthy backends
- `envoy_http_downstream_rq_total` - Total HTTP requests
- `envoy_http_downstream_rq_xx` - HTTP response codes

### Alerting Rules

Example Prometheus rules:

```yaml
groups:
  - name: vpsie_lb
    rules:
      - alert: EnvoyHighErrorRate
        expr: rate(envoy_http_downstream_rq_5xx[5m]) > 0.05
        annotations:
          summary: "High 5xx error rate on {{ $labels.instance }}"

      - alert: EnvoyBackendDown
        expr: envoy_cluster_membership_healthy == 0
        annotations:
          summary: "All backends down for {{ $labels.cluster }}"

      - alert: EnvoyHighMemory
        expr: process_resident_memory_bytes > 2e9
        annotations:
          summary: "Envoy using over 2GB memory"
```

## Logging Configuration

### Log Levels

- **debug**: Verbose debugging information
- **info**: General informational messages (default)
- **warn**: Warning messages
- **error**: Error messages only

### Viewing Logs

```bash
# Agent logs
journalctl -u vpsie-lb-agent -f

# Envoy logs
journalctl -u envoy -f

# Both
journalctl -u vpsie-lb-agent -u envoy -f

# Last 100 lines
journalctl -u vpsie-lb-agent -n 100
```

### Log Rotation

Managed by systemd/journald. Configure in `/etc/systemd/journald.conf`:

```ini
[Journal]
SystemMaxUse=1G
SystemMaxFileSize=100M
MaxRetentionSec=7d
```

## Best Practices

1. **Use HTTPS**: Always use TLS for production traffic
2. **Health Checks**: Configure appropriate health checks for your backends
3. **Timeouts**: Set reasonable timeouts based on your application
4. **Monitoring**: Set up Prometheus + Grafana for visibility
5. **Backups**: Regularly backup `/etc/vpsie-lb/` and `/etc/envoy/`
6. **Updates**: Keep agent and Envoy up to date
7. **Security**: Protect API keys, use strong TLS configuration
8. **Testing**: Test configuration changes in staging first

## Example Configurations

See `examples/` directory for complete configuration examples for common use cases.
