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

echo "Rebooting VM: $VMID"

echo "Info: This will only reboot VMs with a qemu guest agent (you might have to stop it)."

if ! qm status "$VMID" >/dev/null 2>&1; then
    echo "Warning: VM $VMID does not exist."
    exit 0
fi

if qm config "$VMID" | grep -q "template: 1"; then
    echo "Warning: VM $VMID is a template."
fi

if ! qm status "$VMID" | grep -q "status: running"; then
    echo "Warning: VM $VMID is not running."
    exit 0
fi

# Reboot VM, ignore errors
qm reboot "$VMID" || true
