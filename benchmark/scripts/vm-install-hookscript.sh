#!/bin/bash

set -e

VMID="$1"

if [ -z "$VMID" ]; then
    echo "Usage: $0 <vmid> <storage>"
    exit 1
fi

SNIPPET_STORAGE="${PVE_STORAGE_SNIPPETS:-local}"

if ! pvesm status --storage "$SNIPPET_STORAGE" >/dev/null 2>&1; then
    echo "Error: Snipped Storage '$SNIPPET_STORAGE' does not exist or is not active."
    exit 1
fi

echo "Installing hookscript on VM: $VMID with storage: $SNIPPET_STORAGE"

if ! qm status "$VMID" >/dev/null 2>&1; then
    echo "Warning: VM $VMID does not exist."
    exit 0
fi

if ! qm status "$VMID" | grep -q "status: stopped"; then
    echo "Error: VM $VMID is not stopped."
    exit 1
fi

if qm config "$VMID" | grep -q "template: 1"; then
    echo "Warning: VM $VMID is a template."
fi

qm set "${VMID}" --hookscript "${SNIPPET_STORAGE}:snippets/proxmox-cpu-affinity-hook"

exit 0
