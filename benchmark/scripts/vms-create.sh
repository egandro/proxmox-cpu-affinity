#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

. ${SCRIPTDIR}/../config.sh

STORAGE="${STORAGE:-local-zfs}"

USER_SCRIPT=""
if [ -n "$1" ] && [ -f "$1" ]; then
    USER_SCRIPT="$1"
    shift
fi

if [ -n "$1" ]; then
    SOURCE_VMID="$1"
else
    SOURCE_VMID="${BENCHMARK_TEMPLATE_ID:-1001}"
fi

NUM_VMS="${BENCHMARK_NUM_VMS:-2}"
CORES="${BENCHMARK_CORES:-4}"
MEMORY="${BENCHMARK_MEMORY:-1024}"
START_VMID="${BENCHMARK_START_VMID:-200}"

if ! qm status "$SOURCE_VMID" >/dev/null 2>&1; then
    echo "ERROR: Source VM $SOURCE_VMID does not exist."
    exit 1
fi

echo "Starting creation of $NUM_VMS linked clones from VM $SOURCE_VMID..."

for i in $(seq 1 $NUM_VMS); do
    # Calculate the new VMID
    NEW_VMID=$((START_VMID + i - 1))

    echo "[$i/$NUM_VMS] Creating linked clone $NEW_VMID..."

    # Create a linked clone (--full 0)
    qm clone $SOURCE_VMID $NEW_VMID --name "dummy-vm-$NEW_VMID" --full 0

    # Configure resources
    qm set $NEW_VMID --cores $CORES --memory $MEMORY

    if [ -n "$USER_SCRIPT" ]; then
        echo "Running user script $USER_SCRIPT on VM $NEW_VMID with storage $STORAGE..."
        $USER_SCRIPT "$NEW_VMID" "$STORAGE"
    fi
done

echo "All $NUM_VMS VMs created."
