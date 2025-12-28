#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPTDIR}/../.env"
fi

CORES="${BENCHMARK_CORES:-2}"
MEMORY="${BENCHMARK_MEMORY:-1024}"

usage() {
    echo "Usage: $0 [OPTIONS] <vmid> <source_vmid> [user_script]"
    echo "Options:"
    echo "  --force        Stop and destroy VMID if it exists before creating"
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

if [ "$#" -lt 2 ]; then
    usage
fi

VMID="$1"
SOURCE_VMID="$2"
USER_SCRIPT="$3"

if [ "$VMID" = "$SOURCE_VMID" ]; then
    echo "Error: VMID ($VMID) cannot be the same as SOURCE_VMID."
    exit 1
fi

if [ -n "$USER_SCRIPT" ] && [ ! -f "$USER_SCRIPT" ]; then
    echo "Error: User script '$USER_SCRIPT' does not exist."
    exit 1
fi

if qm status "$VMID" >/dev/null 2>&1; then
    if [ "$FORCE_MODE" = true ]; then
        if qm config "$VMID" | grep -q "template: 1"; then
            echo "Error: VM $VMID is a template. Cannot force destroy."
            exit 1
        fi
        echo "Force mode enabled. Destroying existing VM $VMID..."
        qm stop "$VMID" || true
        qm destroy "$VMID" --purge || true
    else
        echo "Error: VM $VMID already exists. Use --force to overwrite."
        exit 1
    fi
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
