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

echo "Stopping VM: $VMID"

if ! qm status "$VMID" >/dev/null 2>&1; then
    echo "Warning: VM $VMID does not exist."
    exit 0
fi

if qm config "$VMID" | grep -q "template: 1"; then
    echo "Error: VM $VMID is a template."
    exit 1
fi

if ! qm status "$VMID" | grep -q "status: running"; then
    echo "Warning: VM $VMID is not running."
    exit 0
fi

# Stopping VM, ignore errors
qm stop --overrule-shutdown 1 "$VMID" || true
