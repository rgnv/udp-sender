#!/bin/bash
# Post-installation script for udp-sender
# Creates udp-senders group and configures permissions

set -e

BINARY_PATH="/usr/bin/udp-sender"
GROUP_NAME="udp-senders"

echo "Configuring udp-sender..."

# Create the udp-senders group if it doesn't exist
if ! getent group "$GROUP_NAME" > /dev/null 2>&1; then
    echo "Creating group: $GROUP_NAME"
    groupadd -r "$GROUP_NAME" || {
        echo "Warning: Failed to create group $GROUP_NAME"
        exit 1
    }
    echo "✓ Group $GROUP_NAME created"
else
    echo "✓ Group $GROUP_NAME already exists"
fi

# Set ownership to root:udp-senders
echo "Setting ownership to root:$GROUP_NAME..."
chown root:"$GROUP_NAME" "$BINARY_PATH" || {
    echo "Warning: Failed to set ownership"
    exit 1
}

# Set permissions to 750 (rwxr-x---)
# Owner (root) can read/write/execute
# Group (udp-senders) can read/execute
# Others cannot access
echo "Setting permissions to 750..."
chmod 750 "$BINARY_PATH" || {
    echo "Warning: Failed to set permissions"
    exit 1
}

# Set CAP_NET_RAW capability
if command -v setcap &> /dev/null; then
    echo "Setting CAP_NET_RAW capability..."
    setcap cap_net_raw+ep "$BINARY_PATH" || {
        echo "Warning: Failed to set CAP_NET_RAW capability."
        echo "You may need to run: sudo setcap cap_net_raw+ep $BINARY_PATH"
        exit 0
    }
    echo "✓ CAP_NET_RAW capability set successfully"
else
    echo "Warning: setcap command not found."
    echo "Raw socket capability could not be configured."
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✓ Installation complete!"
echo ""
echo "IMPORTANT: Only members of the '$GROUP_NAME' group can run udp-sender."
echo ""
echo "To use udp-sender, add your user to the group:"
echo "  sudo usermod -aG $GROUP_NAME \$USER"
echo ""
echo "Then log out and back in (or run: newgrp $GROUP_NAME)"
echo ""
echo "After that, you can run: udp-sender --help"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

exit 0

