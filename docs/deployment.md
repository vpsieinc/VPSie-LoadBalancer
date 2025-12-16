# VPSie Load Balancer Deployment Guide

## Prerequisites

### Build Environment

- Go 1.21+
- Packer 1.9+
- QEMU/KVM
- Make
- Git

### Deployment Environment

- Proxmox VE
- Network connectivity to VPSie API
- Storage for VM images

## Building Images

### Build All Architectures

```bash
make build-images
```

This will create:
- `output/amd64/vpsie-lb-debian-12-amd64-1.0.0.qcow2`
- `output/arm64/vpsie-lb-debian-12-arm64-1.0.0.qcow2`

### Build Specific Architecture

```bash
# AMD64 only
make build-amd64

# ARM64 only
make build-arm64
```

### Build Agent Binary Only

```bash
# For local architecture
make build-agent

# For specific architecture
make build-agent GOARCH=arm64
```

## Deploying to Proxmox

### Upload Image

```bash
# Copy image to Proxmox host
scp output/amd64/vpsie-lb-debian-12-amd64-1.0.0.qcow2 root@proxmox:/var/lib/vz/images/
```

### Create VM

```bash
# SSH to Proxmox host
ssh root@proxmox

# Create VM
VM_ID=100
VM_NAME="vpsie-lb-01"

qm create $VM_ID \
    --name $VM_NAME \
    --memory 4096 \
    --cores 4 \
    --net0 virtio,bridge=vmbr0 \
    --serial0 socket \
    --vga serial0

# Import disk
qm importdisk $VM_ID /var/lib/vz/images/vpsie-lb-debian-12-amd64-1.0.0.qcow2 local-lvm

# Attach disk
qm set $VM_ID --scsihw virtio-scsi-pci --scsi0 local-lvm:vm-$VM_ID-disk-0

# Set boot disk
qm set $VM_ID --boot c --bootdisk scsi0

# Set BIOS
qm set $VM_ID --bios ovmf

# Optional: Set CPU type
qm set $VM_ID --cpu host

# Start VM
qm start $VM_ID
```

### Access VM

```bash
# Get IP address (check Proxmox console or DHCP)
qm guest exec $VM_ID -- ip addr show

# SSH to VM
ssh root@<vm-ip>
```

## Initial Configuration

### 1. Set API Key

```bash
# On the VM
echo "your-vpsie-api-key" > /etc/vpsie-lb/api-key
chmod 600 /etc/vpsie-lb/api-key
```

### 2. Configure Agent

Edit `/etc/vpsie-lb/agent.yaml`:

```yaml
vpsie:
  api_url: https://api.vpsie.com/v1
  api_key_file: /etc/vpsie-lb/api-key
  loadbalancer_id: lb-your-id-here  # Get from VPSie
  poll_interval: 30s

envoy:
  config_path: /etc/envoy/dynamic
  admin_address: 127.0.0.1:9901
  binary_path: /usr/bin/envoy

logging:
  level: info
  format: json
```

### 3. Start Services

```bash
# Start Envoy
systemctl start envoy
systemctl status envoy

# Start Agent
systemctl start vpsie-lb-agent
systemctl status vpsie-lb-agent

# Enable on boot
systemctl enable envoy
systemctl enable vpsie-lb-agent
```

### 4. Verify Operation

```bash
# Check agent logs
journalctl -u vpsie-lb-agent -f

# Check Envoy logs
journalctl -u envoy -f

# Check Envoy admin interface
curl http://127.0.0.1:9901/stats
curl http://127.0.0.1:9901/clusters
curl http://127.0.0.1:9901/listeners
```

## Network Configuration

### Firewall Rules

```bash
# Allow load balancer ports (example)
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

# Save rules
iptables-save > /etc/iptables/rules.v4
```

### Static IP Configuration

Edit `/etc/network/interfaces`:

```bash
auto eth0
iface eth0 inet static
    address 192.168.1.100
    netmask 255.255.255.0
    gateway 192.168.1.1
    dns-nameservers 8.8.8.8 8.8.4.4
```

Restart networking:

```bash
systemctl restart networking
```

## Monitoring Setup

### Prometheus Scraping

Add to Prometheus config:

```yaml
scrape_configs:
  - job_name: 'vpsie-lb'
    static_configs:
      - targets: ['<lb-ip>:9901']
    metrics_path: /stats/prometheus
```

### Grafana Dashboard

Import dashboard from `monitoring/grafana/dashboard.json`

## Upgrading

### Agent Upgrade

```bash
# Build new agent binary
make build-agent

# Copy to VM
scp build/vpsie-lb-agent-amd64 root@<vm-ip>:/tmp/

# On VM
systemctl stop vpsie-lb-agent
mv /tmp/vpsie-lb-agent-amd64 /usr/local/bin/vpsie-lb-agent
chmod +x /usr/local/bin/vpsie-lb-agent
systemctl start vpsie-lb-agent
```

### Full Image Upgrade

```bash
# Build new image
make build-amd64 VERSION=1.1.0

# Create new VM from new image
# Migrate traffic
# Decommission old VM
```

## Troubleshooting

### Agent Not Starting

```bash
# Check configuration
cat /etc/vpsie-lb/agent.yaml

# Check API key
cat /etc/vpsie-lb/api-key

# Check logs
journalctl -u vpsie-lb-agent -n 100

# Test API connectivity
curl -H "Authorization: Bearer $(cat /etc/vpsie-lb/api-key)" \
    https://api.vpsie.com/v1/loadbalancers/<your-lb-id>
```

### Envoy Not Starting

```bash
# Check configuration syntax
envoy --mode validate -c /etc/envoy/bootstrap.yaml

# Check logs
journalctl -u envoy -n 100

# Check file permissions
ls -la /etc/envoy/
ls -la /var/log/envoy/
```

### Configuration Not Updating

```bash
# Check agent logs for errors
journalctl -u vpsie-lb-agent -f

# Manually trigger sync
systemctl restart vpsie-lb-agent

# Check generated configs
cat /etc/envoy/dynamic/listeners.yaml
cat /etc/envoy/dynamic/clusters.yaml
```

### High Memory Usage

```bash
# Check Envoy memory
ps aux | grep envoy

# Check for memory leaks in stats
curl http://127.0.0.1:9901/memory

# Restart Envoy if needed
systemctl restart envoy
```

## Backup and Recovery

### Backup Configuration

```bash
tar -czf vpsie-lb-backup-$(date +%Y%m%d).tar.gz \
    /etc/vpsie-lb/ \
    /etc/envoy/
```

### Restore Configuration

```bash
tar -xzf vpsie-lb-backup-20250101.tar.gz -C /
systemctl restart vpsie-lb-agent
```

## Performance Tuning

See [Configuration Guide](configuration.md) for detailed tuning options.
