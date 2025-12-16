#!/bin/bash
set -e

echo "=== Setting up systemd services ==="

# Copy systemd service files (these should be uploaded during provisioning)
# For now, we'll create them inline

# Envoy service
cat > /etc/systemd/system/envoy.service <<'EOF'
[Unit]
Description=Envoy Proxy
Documentation=https://www.envoyproxy.io/
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=envoy
Group=envoy
ExecStart=/usr/bin/envoy -c /etc/envoy/bootstrap.yaml --service-cluster vpsie-lb
Restart=always
RestartSec=5
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF

# Agent service
cat > /etc/systemd/system/vpsie-lb-agent.service <<'EOF'
[Unit]
Description=VPSie Load Balancer Control Plane Agent
After=network-online.target envoy.service
Wants=network-online.target
Requires=envoy.service

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/vpsie-lb-agent --config /etc/vpsie-lb/agent.yaml
Restart=always
RestartSec=10
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd
systemctl daemon-reload

# Enable services (don't start them yet)
systemctl enable envoy.service
systemctl enable vpsie-lb-agent.service

echo "=== Systemd setup complete ==="
