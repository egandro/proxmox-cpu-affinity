#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPTDIR}/../.env"
fi

SOURCE_VMID="${BENCHMARK_TEMPLATE_ID:-1001}"

CORES="${BENCHMARK_CORES:-2}"
MEMORY="${BENCHMARK_MEMORY:-1024}"

if [ -z "$1" ]; then
    echo "Usage: $0 <vmid> [user_script]"
    exit 1
fi
VMID="$1"

USER_SCRIPT=""
if [ -n "$2" ] && [ -f "$2" ]; then
    USER_SCRIPT="$2"
    shift
fi

if ! qm status "$SOURCE_VMID" >/dev/null 2>&1; then
    echo "ERROR: Source VM $SOURCE_VMID does not exist."
    exit 1
fi

echo "Creating VM: $VMID from template: $SOURCE_VMID"

# Create a linked clone (--full 0)
qm clone "$SOURCE_VMID" "$VMID" --name "dummy-vm-$VMID" --full 0

# Configure resources
qm set "$VMID" --cores "$CORES" --memory "$MEMORY"

if [ -n "$USER_SCRIPT" ]; then
    echo "Running user script $USER_SCRIPT on VM $VMID with storage $PVE_STORAGE..."
    $USER_SCRIPT "$VMID" "$PVE_STORAGE"
fi
