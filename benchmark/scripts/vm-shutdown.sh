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

echo "Shutdown VM: $VMID"

echo "Warning: This will only shutdown VMs with a qemu guest agent."

if ! qm status "$VMID" >/dev/null 2>&1; then
    echo "Warning: VM $VMID does not exist."
    exit 0
fi

# Shutdown VM, ignore errors
qm shutdown "$VMID" || true
