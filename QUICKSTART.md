# VPSie Load Balancer - Quick Start Guide

## Prerequisites

- Go 1.21+
- Packer 1.9+
- QEMU/KVM (for building images)
- Proxmox VE (for deployment)
- VPSie Account with API access

## 1. Build the Project

### Build Agent Binary

```bash
# Clone the repository
cd vpsie-loadbalancer

# Download dependencies
make deps

# Build agent for Linux amd64
make build-agent GOOS=linux GOARCH=amd64

# The binary will be in: build/vpsie-lb-agent-amd64
```

### Build VM Images

```bash
# Build both amd64 and arm64 images
make build-images

# Or build specific architecture
make build-amd64
make build-arm64

# Images will be in: output/amd64/ and output/arm64/
```

## 2. Deploy to Proxmox

### Upload Image

```bash
scp output/amd64/vpsie-lb-debian-12-amd64-1.0.0.qcow2 root@proxmox:/var/lib/vz/images/
```

### Create VM

```bash
ssh root@proxmox

# Create VM
qm create 100 --name vpsie-lb-01 --memory 4096 --cores 4 --net0 virtio,bridge=vmbr0

# Import disk
qm importdisk 100 /var/lib/vz/images/vpsie-lb-debian-12-amd64-1.0.0.qcow2 local-lvm

# Attach disk
qm set 100 --scsihw virtio-scsi-pci --scsi0 local-lvm:vm-100-disk-0

# Set boot disk
qm set 100 --boot c --bootdisk scsi0

# Start VM
qm start 100
```

## 3. Configure Load Balancer

### SSH to VM

```bash
# Find VM IP (from Proxmox console or DHCP)
ssh root@<vm-ip>
```

### Set VPSie API Key

```bash
echo "your-vpsie-api-key-here" > /etc/vpsie-lb/api-key
chmod 600 /etc/vpsie-lb/api-key
```

### Configure Agent

Edit `/etc/vpsie-lb/agent.yaml`:

```bash
vim /etc/vpsie-lb/agent.yaml
```

Update these fields:
```yaml
vpsie:
  loadbalancer_id: lb-your-id-from-vpsie  # REQUIRED
  api_url: https://api.vpsie.com/v1
```

## 4. Start Services

```bash
# Start Envoy
systemctl start envoy
systemctl enable envoy

# Start Agent
systemctl start vpsie-lb-agent
systemctl enable vpsie-lb-agent

# Check status
systemctl status envoy
systemctl status vpsie-lb-agent
```

## 5. Verify Operation

### Check Logs

```bash
# Watch agent logs
journalctl -u vpsie-lb-agent -f

# Check for successful config sync
# You should see: "Configuration sync completed successfully"
```

### Check Envoy Stats

```bash
# Test Envoy admin interface
curl http://127.0.0.1:9901/stats

# Check clusters (backends)
curl http://127.0.0.1:9901/clusters

# Check listeners
curl http://127.0.0.1:9901/listeners
```

### Test Load Balancing

```bash
# From another machine, test your load balancer
curl http://<lb-ip>

# For HTTPS
curl https://<lb-ip>

# Check if traffic reaches backends
```

## 6. Monitor

### Prometheus Metrics

Configure Prometheus to scrape: `http://<lb-ip>:9901/stats/prometheus`

### View Metrics

```bash
# On the load balancer VM
curl http://127.0.0.1:9901/stats/prometheus | grep envoy_cluster
```

## Troubleshooting

### Agent Won't Start

```bash
# Check configuration
cat /etc/vpsie-lb/agent.yaml

# Check API key
cat /etc/vpsie-lb/api-key

# Test API connectivity
curl -H "Authorization: Bearer $(cat /etc/vpsie-lb/api-key)" \
     https://api.vpsie.com/v1/loadbalancers/<your-lb-id>

# Check detailed logs
journalctl -u vpsie-lb-agent -n 100 --no-pager
```

### Envoy Won't Start

```bash
# Validate Envoy config
envoy --mode validate -c /etc/envoy/bootstrap.yaml

# Check permissions
ls -la /etc/envoy/
ls -la /var/log/envoy/

# Check Envoy logs
journalctl -u envoy -n 100 --no-pager
```

### Configuration Not Syncing

```bash
# Force restart agent
systemctl restart vpsie-lb-agent

# Watch logs for errors
journalctl -u vpsie-lb-agent -f

# Check generated configs
cat /etc/envoy/dynamic/listeners.yaml
cat /etc/envoy/dynamic/clusters.yaml
```

## Next Steps

1. **Configure VPSie**: Set up your load balancer configuration in VPSie dashboard
2. **Add Backends**: Add backend servers to your load balancer
3. **Configure Health Checks**: Set up appropriate health checks
4. **Set Up Monitoring**: Configure Prometheus and Grafana
5. **Enable HTTPS**: Add TLS certificates for HTTPS support

## Documentation

- [Architecture](docs/architecture.md) - System architecture overview
- [Deployment](docs/deployment.md) - Detailed deployment guide
- [Configuration](docs/configuration.md) - Configuration reference

## Support

For issues or questions:
- GitHub Issues: https://github.com/vpsie/vpsie-loadbalancer/issues
- Documentation: https://docs.vpsie.com/
