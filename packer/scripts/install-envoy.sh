#!/bin/bash
set -e

echo "=== Installing Envoy Proxy ==="

# Detect architecture
ARCH=$(dpkg --print-architecture)
echo "Architecture: $ARCH"

# Add Envoy repository (using Tetrate GetEnvoy)
curl -sLo /tmp/envoy-gpg.key 'https://getenvoy.io/gpg'
gpg --dearmor -o /usr/share/keyrings/getenvoy-keyring.gpg /tmp/envoy-gpg.key
rm -f /tmp/envoy-gpg.key

echo "deb [arch=$ARCH signed-by=/usr/share/keyrings/getenvoy-keyring.gpg] https://deb.dl.getenvoy.io/public/deb/debian $(lsb_release -cs) main" | \
    tee /etc/apt/sources.list.d/getenvoy.list

# Update and install Envoy
apt-get update
apt-get install -y getenvoy-envoy || {
    echo "Warning: GetEnvoy installation failed, trying alternative method..."

    # Alternative: Install from Envoy official releases
    ENVOY_VERSION="1.28.0"
    if [ "$ARCH" = "amd64" ]; then
        wget -O /tmp/envoy.tar.gz "https://github.com/envoyproxy/envoy/releases/download/v${ENVOY_VERSION}/envoy-${ENVOY_VERSION}-linux-x86_64"
    elif [ "$ARCH" = "arm64" ]; then
        wget -O /tmp/envoy.tar.gz "https://github.com/envoyproxy/envoy/releases/download/v${ENVOY_VERSION}/envoy-${ENVOY_VERSION}-linux-aarch64"
    fi

    mv /tmp/envoy.tar.gz /usr/bin/envoy
    chmod +x /usr/bin/envoy
}

# Verify installation
envoy --version

# Create envoy user
useradd -r -s /bin/false envoy || true
chown -R envoy:envoy /etc/envoy
chown -R envoy:envoy /var/log/envoy

echo "=== Envoy installation complete ==="
