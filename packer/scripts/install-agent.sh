#!/bin/bash
set -e

echo "=== Installing VPSie Load Balancer Agent ==="

# Install Go (for building the agent)
GO_VERSION="1.21.5"
ARCH=$(dpkg --print-architecture)

if [ "$ARCH" = "amd64" ]; then
    GO_ARCH="amd64"
elif [ "$ARCH" = "arm64" ]; then
    GO_ARCH="arm64"
fi

wget -O /tmp/go.tar.gz "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
rm -rf /usr/local/go
tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz

export PATH=$PATH:/usr/local/go/bin

# Copy agent source (this assumes the source is uploaded)
mkdir -p /tmp/agent-build
cd /tmp/agent-build

# In production, you would:
# 1. Copy the source code to the VM
# 2. Build the agent binary
# For now, we'll create a placeholder

# Build agent (assuming source is available)
# go build -o vpsie-lb-agent github.com/vpsie/vpsie-loadbalancer/cmd/agent

# Install agent binary
# install -m 755 vpsie-lb-agent /usr/local/bin/vpsie-lb-agent

# For now, create a note that the binary needs to be provided
cat > /usr/local/bin/vpsie-lb-agent <<'EOF'
#!/bin/bash
echo "VPSie LB Agent placeholder - replace with actual binary"
EOF
chmod +x /usr/local/bin/vpsie-lb-agent

# Clean up Go installation (optional, to reduce image size)
# rm -rf /usr/local/go

echo "=== Agent installation complete ==="
