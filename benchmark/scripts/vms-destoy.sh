#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

. ${SCRIPTDIR}/../config.sh

NUM_VMS="${BENCHMARK_NUM_VMS:-2}"
START_VMID="${BENCHMARK_START_VMID:-200}"

echo "Starting destruction of $NUM_VMS dummy VMs starting from ID $START_VMID..."

for i in $(seq 1 $NUM_VMS); do
    # Calculate the VMID
    VMID=$((START_VMID + i - 1))

    echo "[$i/$NUM_VMS] Processing VM $VMID..."

    # Stop the VM (hard stop) to ensure it can be destroyed
    qm stop $VMID || true

    # Destroy the VM and purge disks
    qm destroy $VMID --purge || true
done
