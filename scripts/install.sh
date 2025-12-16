#!/bin/bash
set -e

# Installation script for VPSie Load Balancer
# This script runs on first boot of the VM

echo "=== VPSie Load Balancer First Boot Setup ==="

# Generate SSH host keys if they don't exist
if [ ! -f /etc/ssh/ssh_host_rsa_key ]; then
    echo "Generating SSH host keys..."
    ssh-keygen -A
fi

# Create required directories
mkdir -p /etc/vpsie-lb
mkdir -p /etc/envoy/dynamic
mkdir -p /var/log/envoy
mkdir -p /var/run

# Set permissions
chown -R envoy:envoy /etc/envoy
chown -R envoy:envoy /var/log/envoy
chmod 700 /etc/vpsie-lb

# Configure network (if needed)
# This would be customized based on VPSie's network setup

# Create placeholder API key file if it doesn't exist
if [ ! -f /etc/vpsie-lb/api-key ]; then
    echo "CHANGE-ME-API-KEY" > /etc/vpsie-lb/api-key
    chmod 600 /etc/vpsie-lb/api-key
    echo "WARNING: API key not configured! Edit /etc/vpsie-lb/api-key"
fi

# Create default agent config if it doesn't exist
if [ ! -f /etc/vpsie-lb/agent.yaml ]; then
    cat > /etc/vpsie-lb/agent.yaml <<'EOF'
vpsie:
  api_url: https://api.vpsie.com/v1
  api_key_file: /etc/vpsie-lb/api-key
  loadbalancer_id: CHANGE-ME
  poll_interval: 30s

envoy:
  config_path: /etc/envoy/dynamic
  admin_address: 127.0.0.1:9901
  binary_path: /usr/bin/envoy

logging:
  level: info
  format: json
EOF
    echo "Default agent configuration created at /etc/vpsie-lb/agent.yaml"
    echo "Please update the loadbalancer_id in the configuration"
fi

# Create bootstrap config if it doesn't exist
if [ ! -f /etc/envoy/bootstrap.yaml ]; then
    cat > /etc/envoy/bootstrap.yaml <<'EOF'
node:
  id: vpsie-lb-node
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

layered_runtime:
  layers:
    - name: static_layer
      static_layer:
        overload:
          global_downstream_max_connections: 50000
EOF
fi

# Create empty dynamic configs
touch /etc/envoy/dynamic/listeners.yaml
touch /etc/envoy/dynamic/clusters.yaml
chown envoy:envoy /etc/envoy/dynamic/*.yaml

echo "=== Setup complete ==="
echo ""
echo "Next steps:"
echo "1. Edit /etc/vpsie-lb/api-key with your VPSie API key"
echo "2. Edit /etc/vpsie-lb/agent.yaml and set your loadbalancer_id"
echo "3. Start services:"
echo "   systemctl start envoy"
echo "   systemctl start vpsie-lb-agent"
echo "4. Check status:"
echo "   systemctl status envoy"
echo "   systemctl status vpsie-lb-agent"
echo "   journalctl -u vpsie-lb-agent -f"
