#!/bin/bash
set -e

echo "=== Provisioning VPSie Load Balancer Base System ==="

# Update system
apt-get update
apt-get upgrade -y

# Install base packages
apt-get install -y \
    curl \
    wget \
    gnupg \
    lsb-release \
    ca-certificates \
    apt-transport-https \
    software-properties-common \
    sudo \
    vim \
    htop \
    net-tools \
    iptables \
    systemd \
    dbus \
    qemu-guest-agent

# Configure system limits
cat > /etc/sysctl.d/99-vpsie-lb.conf <<EOF
# Network performance tuning
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
EOF

# Apply sysctl settings
sysctl -p /etc/sysctl.d/99-vpsie-lb.conf

# Configure file limits
cat > /etc/security/limits.d/99-vpsie-lb.conf <<EOF
* soft nofile 65536
* hard nofile 65536
* soft nproc 4096
* hard nproc 4096
EOF

# Create directories
mkdir -p /etc/vpsie-lb
mkdir -p /etc/envoy/dynamic
mkdir -p /var/log/envoy
mkdir -p /var/run

echo "=== Base provisioning complete ==="
