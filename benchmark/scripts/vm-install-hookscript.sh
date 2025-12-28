#!/bin/bash

set -e

VMID="$1"
STORAGE="$2"

if [ -z "$VMID" ] || [ -z "$STORAGE" ]; then
    echo "Usage: $0 <vmid> <storage>"
    exit 1
fi

echo "Installing hookscript on VM: $VMID with storage: $STORAGE"

if ! pvesm status --storage "$STORAGE" >/dev/null 2>&1; then
    echo "Error: Storage '$STORAGE' does not exist or is not active."
    exit 1
fi

if ! qm status "$VMID" >/dev/null 2>&1; then
    echo "Warning: VM $VMID does not exist."
    exit 0
fi

if ! qm status "$VMID" | grep -q "status: stopped"; then
    echo "Error: VM $VMID is not stopped."
    exit 1
fi

qm set "${VMID}" --hookscript "${STORAGE}:snippets/proxmox-cpu-affinity-hook"

exit 0
