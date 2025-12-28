#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

. ${SCRIPTDIR}/../config.sh

STORAGE="${STORAGE:-local-zfs}"

if [ ! -f "$1" ]; then
    echo "Error: User script required."
    exit 1
fi

USER_SCRIPT="$1"

NUM_VMS="${BENCHMARK_NUM_VMS:-2}"
START_VMID="${BENCHMARK_START_VMID:-200}"

echo "Modifying $NUM_VMS dummy VMs starting from ID $START_VMID..."

for i in $(seq 1 $NUM_VMS); do
    # Calculate the VMID
    VMID=$((START_VMID + i - 1))

    echo "[$i/$NUM_VMS] Processing VM $VMID..."

    if ! qm status $VMID | grep -q "status: stopped"; then
        echo "Error: VM $VMID is not stopped."
        exit 1
    fi

    echo "Running user script $USER_SCRIPT on VM $VMID with storage $PVE_STORAGE..."
    $USER_SCRIPT "$VMID" "$PVE_STORAGE"
done
