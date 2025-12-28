#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPTDIR}/../.env"
fi

STORAGE="${PVE_STORAGE:-local-zfs}"

if [ -z "$1" ] && [ -z "$2" ]; then
    echo "Usage: $0 <vmid> <modify_script>"
    exit 1
fi

VMID="$1"
MODIFY_SCRIPT="$2"

if ! qm status "$VMID" | grep -q "status: stopped"; then
    echo "Error: VM $VMID is not stopped."
    exit 1
fi

echo "Running modification script $MODIFY_SCRIPT on VM: $VMID (storage: $STORAGE)"

$MODIFY_SCRIPT "$VMID" "$STORAGE"
