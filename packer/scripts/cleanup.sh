#!/bin/bash
set -e

echo "=== Cleaning up image ==="

# Remove unnecessary packages
apt-get autoremove -y
apt-get autoclean -y

# Clear package cache
apt-get clean
rm -rf /var/lib/apt/lists/*

# Clear logs
find /var/log -type f -exec truncate -s 0 {} \;

# Clear temporary files
rm -rf /tmp/*
rm -rf /var/tmp/*

# Clear bash history
history -c
rm -f /root/.bash_history

# Clear SSH host keys (will be regenerated on first boot)
rm -f /etc/ssh/ssh_host_*

# Zero out free space to improve compression
dd if=/dev/zero of=/EMPTY bs=1M || true
rm -f /EMPTY

sync

echo "=== Cleanup complete ==="
