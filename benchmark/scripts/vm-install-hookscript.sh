#!/bin/bash

set -e

VMID="$1"
STORAGE="$2"

if [ -z "$VMID" ] || [ -z "$STORAGE" ]; then
    echo "Usage: $0 <vmid> <storage>"
    exit 1
fi

if [ -z "$1" ]; then
    echo "Usage: $0 <vmid>"
    exit 1
fi
VMID="$1"

if ! qm status "$VMID" | grep -q "status: stopped"; then
    echo "Error: VM $VMID is not stopped."
    exit 1
fi

echo "Installing hookscript on VM: $VMID with storage: $STORAGE"

qm set "${VMID}" --hookscript "${STORAGE}:snippets/proxmox-cpu-affinity-hook"

exit 0
