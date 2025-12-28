#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPTDIR}/../.env"
fi

if [ -z "$1" ]; then
    echo "Usage: $0 <vmid>"
    exit 1
fi
VMID="$1"

echo "Destroying VM: $VMID"

# Stop the VM (hard stop) to ensure it can be destroyed
qm stop "$VMID" || true

# Destroy the VM and purge disks
qm destroy "$VMID" --purge || true
