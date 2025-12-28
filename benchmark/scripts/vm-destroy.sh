#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPTDIR}/../.env"
fi

usage() {
    echo "Usage: $0 [OPTIONS] <vmid>"
    echo "Options:"
    echo "  --force        Force delete linked clones"
    exit 1
}

FORCE_MODE=false
POSITIONAL_ARGS=()

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --force) FORCE_MODE=true ;;
        -h|--help) usage ;;
        *) POSITIONAL_ARGS+=("$1") ;;
    esac
    shift
done
set -- "${POSITIONAL_ARGS[@]}"

if [ "$#" -lt 1 ]; then
    usage
fi
VMID="$1"

echo "Destroying VM: $VMID"

if ! qm status "$VMID" >/dev/null 2>&1; then
    echo "Warning: VM $VMID does not exist."
    exit 0
fi

if qm config "$VMID" | grep -q "template: 1"; then
    echo "Warning: VM $VMID is a template."
fi

# Stop the VM (hard stop) to ensure it can be destroyed
qm stop --overrule-shutdown 1 "$VMID" || true

# Check for linked clones
CLONES=$(grep -l "base-${VMID}-disk" /etc/pve/qemu-server/*.conf 2>/dev/null | grep -v "/${VMID}.conf" || true)
if [ -n "$CLONES" ]; then
    if [ "$FORCE_MODE" = true ]; then
        echo "Found linked clones. Forcing removal..."
        for conf in $CLONES; do
            CLONE_ID=$(basename "$conf" .conf)
            echo "Destroying linked clone $CLONE_ID..."
            if qm status "$CLONE_ID" | grep -q "status: running"; then
                qm stop "$CLONE_ID" --overrule-shutdown 1 &>/dev/null || true
            fi
            qm destroy "$CLONE_ID"
        done
    else
        echo "Error: Linked clones found for VM $VMID. Use --force to delete them."
        exit 1
    fi
fi

# Destroy the VM and purge disks
qm destroy "$VMID" --purge || true
