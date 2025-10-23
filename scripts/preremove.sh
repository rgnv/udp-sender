#!/bin/bash
# Pre-removal script for udp-sender
# Optionally removes the udp-senders group if empty

set -e

GROUP_NAME="udp-senders"

echo "Removing udp-sender..."

# Check if the group exists
if getent group "$GROUP_NAME" > /dev/null 2>&1; then
    # Check if any users are in the group (excluding the implicit group membership)
    GROUP_MEMBERS=$(getent group "$GROUP_NAME" | cut -d: -f4)
    
    if [ -z "$GROUP_MEMBERS" ]; then
        echo "Removing empty group: $GROUP_NAME"
        groupdel "$GROUP_NAME" 2>/dev/null || {
            echo "Note: Could not remove group $GROUP_NAME (may have active processes)"
        }
    else
        echo "Note: Group $GROUP_NAME has members, not removing"
        echo "To manually remove: sudo groupdel $GROUP_NAME"
    fi
fi

exit 0

