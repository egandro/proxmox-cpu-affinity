#!/bin/bash

set -e

VMID="$1"
STORAGE="$2"

if [ -z "$VMID" ] || [ -z "$STORAGE" ]; then
    echo "Usage: $0 <vmid> <storage>"
    exit 1
fi

qm set ${VMID} --hookscript ${STORAGE}:snippets/proxmox-cpu-affinity-hook

exit 0
