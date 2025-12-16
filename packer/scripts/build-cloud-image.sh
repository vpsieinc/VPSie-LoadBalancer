#!/bin/bash
set -e

# Build VPSie Load Balancer image from Debian cloud image
# Uses virt-customize (libguestfs) - no VM boot required

ARCH="${1:-amd64}"
VERSION="${2:-0.0.0}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKER_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="${PACKER_DIR}/../build"
OUTPUT_DIR="${PACKER_DIR}/output/${ARCH}"

# Debian cloud image URL
DEBIAN_IMAGE_URL="https://cloud.debian.org/images/cloud/trixie/daily/latest/debian-13-nocloud-${ARCH}-daily.qcow2"
DEBIAN_IMAGE_NAME="debian-13-nocloud-${ARCH}-daily.qcow2"

echo "=== Building VPSie Load Balancer Image ==="
echo "Architecture: ${ARCH}"
echo "Version: ${VERSION}"

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Download Debian cloud image if not cached
CACHE_DIR="${HOME}/.cache/vpsie-lb"
mkdir -p "${CACHE_DIR}"

if [ ! -f "${CACHE_DIR}/${DEBIAN_IMAGE_NAME}" ]; then
    echo "=== Downloading Debian cloud image ==="
    curl -L -o "${CACHE_DIR}/${DEBIAN_IMAGE_NAME}" "${DEBIAN_IMAGE_URL}"
else
    echo "=== Using cached Debian cloud image ==="
fi

# Copy base image to output
OUTPUT_IMAGE="${OUTPUT_DIR}/vpsie-lb-debian-13-${ARCH}-${VERSION}.qcow2"
cp "${CACHE_DIR}/${DEBIAN_IMAGE_NAME}" "${OUTPUT_IMAGE}"

# Resize the image to 10GB
echo "=== Resizing image to 10GB ==="
qemu-img resize "${OUTPUT_IMAGE}" 10G

# Check if agent binary exists
AGENT_BINARY="${BUILD_DIR}/vpsie-lb-agent-${ARCH}"
if [ ! -f "${AGENT_BINARY}" ]; then
    echo "ERROR: Agent binary not found at ${AGENT_BINARY}"
    exit 1
fi

echo "=== Customizing image with virt-customize ==="

# Set libguestfs backend to direct (avoids passt networking issues)
export LIBGUESTFS_BACKEND=direct

# Create a temporary directory for files to copy
TEMP_DIR=$(mktemp -d)
trap "rm -rf ${TEMP_DIR}" EXIT

# Copy agent binary
cp "${AGENT_BINARY}" "${TEMP_DIR}/vpsie-lb-agent"
chmod +x "${TEMP_DIR}/vpsie-lb-agent"

# Create systemd service files
cat > "${TEMP_DIR}/vpsie-lb-agent.service" <<'EOF'
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

cat > "${TEMP_DIR}/envoy.service" <<'EOF'
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

# Create sysctl config
cat > "${TEMP_DIR}/99-vpsie-lb.conf" <<'EOF'
# Network performance tuning
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 8192
net.ipv4.ip_local_port_range = 1024 65535
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 30
net.core.netdev_max_backlog = 5000

# Memory
vm.swappiness = 10
EOF

# Create limits config
cat > "${TEMP_DIR}/99-vpsie-lb-limits.conf" <<'EOF'
* soft nofile 65536
* hard nofile 65536
* soft nproc 4096
* hard nproc 4096
EOF

# Create firstboot script for all setup operations
# This approach works for both amd64 and arm64 (cross-architecture compatible)
cat > "${TEMP_DIR}/vpsie-firstboot.sh" <<'FIRSTBOOT'
#!/bin/bash
set -e

STAGING_DIR="/root/vpsie-staging"

# Create directories
mkdir -p /etc/vpsie-lb /etc/envoy/dynamic /var/log/envoy /usr/local/bin

# Move staged files to final locations
if [ -d "${STAGING_DIR}" ]; then
    # Move agent binary
    if [ -f "${STAGING_DIR}/vpsie-lb-agent" ]; then
        mv "${STAGING_DIR}/vpsie-lb-agent" /usr/local/bin/
        chmod +x /usr/local/bin/vpsie-lb-agent
    fi

    # Move systemd service files
    for service in vpsie-lb-agent.service envoy.service; do
        if [ -f "${STAGING_DIR}/${service}" ]; then
            mv "${STAGING_DIR}/${service}" /etc/systemd/system/
        fi
    done

    # Move sysctl config
    if [ -f "${STAGING_DIR}/99-vpsie-lb.conf" ]; then
        mv "${STAGING_DIR}/99-vpsie-lb.conf" /etc/sysctl.d/
    fi

    # Move limits config
    if [ -f "${STAGING_DIR}/99-vpsie-lb-limits.conf" ]; then
        mv "${STAGING_DIR}/99-vpsie-lb-limits.conf" /etc/security/limits.d/
    fi

    # Cleanup staging directory
    rm -rf "${STAGING_DIR}"
fi

# Wait for network
sleep 10

# Update and install packages
apt-get update
apt-get install -y curl wget gnupg ca-certificates apt-transport-https sudo vim htop net-tools iptables qemu-guest-agent

# Install Envoy
curl -sL https://func-e.io/install.sh | bash -s -- -b /usr/local/bin
/usr/local/bin/func-e use 1.28.0
cp ~/.func-e/versions/1.28.0/bin/envoy /usr/bin/envoy

# Create envoy user
useradd --system --no-create-home --shell /bin/false envoy || true

# Apply sysctl settings
sysctl -p /etc/sysctl.d/99-vpsie-lb.conf || true

# Reload systemd and enable services
systemctl daemon-reload
systemctl enable vpsie-lb-agent.service
systemctl enable envoy.service
systemctl enable qemu-guest-agent.service
systemctl start qemu-guest-agent.service

# Cleanup
apt-get clean
rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Mark firstboot as done
touch /var/lib/vpsie-firstboot-done
FIRSTBOOT
chmod +x "${TEMP_DIR}/vpsie-firstboot.sh"

# Create firstboot service
cat > "${TEMP_DIR}/vpsie-firstboot.service" <<'EOF'
[Unit]
Description=VPSie First Boot Setup
After=network-online.target
Wants=network-online.target
ConditionPathExists=!/var/lib/vpsie-firstboot-done

[Service]
Type=oneshot
ExecStart=/usr/local/bin/vpsie-firstboot.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

# Create staging directory structure
STAGING_DIR="${TEMP_DIR}/staging"
mkdir -p "${STAGING_DIR}"

# Copy all files to staging directory
cp "${TEMP_DIR}/vpsie-lb-agent" "${STAGING_DIR}/"
cp "${TEMP_DIR}/vpsie-lb-agent.service" "${STAGING_DIR}/"
cp "${TEMP_DIR}/envoy.service" "${STAGING_DIR}/"
cp "${TEMP_DIR}/99-vpsie-lb.conf" "${STAGING_DIR}/"
cp "${TEMP_DIR}/99-vpsie-lb-limits.conf" "${STAGING_DIR}/"

# Run virt-customize (no network, no run-command - cross-architecture compatible)
# Files are staged to /root/vpsie-staging and moved by firstboot script
virt-customize -a "${OUTPUT_IMAGE}" \
    --no-network \
    --mkdir /root/vpsie-staging \
    --copy-in "${STAGING_DIR}/vpsie-lb-agent:/root/vpsie-staging/" \
    --copy-in "${STAGING_DIR}/vpsie-lb-agent.service:/root/vpsie-staging/" \
    --copy-in "${STAGING_DIR}/envoy.service:/root/vpsie-staging/" \
    --copy-in "${STAGING_DIR}/99-vpsie-lb.conf:/root/vpsie-staging/" \
    --copy-in "${STAGING_DIR}/99-vpsie-lb-limits.conf:/root/vpsie-staging/" \
    --copy-in "${TEMP_DIR}/vpsie-firstboot.service:/etc/systemd/system/" \
    --firstboot "${TEMP_DIR}/vpsie-firstboot.sh"

# Generate checksum
echo "=== Generating checksum ==="
CHECKSUM_FILE="${OUTPUT_DIR}/vpsie-lb-debian-13-${ARCH}-${VERSION}.checksum"
sha256sum "${OUTPUT_IMAGE}" | awk '{print $1}' > "${CHECKSUM_FILE}"

echo "=== Build complete ==="
echo "Image: ${OUTPUT_IMAGE}"
echo "Checksum: ${CHECKSUM_FILE}"
