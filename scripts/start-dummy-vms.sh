#!/bin/bash

set -e

NUM_VMS=10
START_VMID=200

echo "Starting $NUM_VMS dummy VMs starting from ID $START_VMID..."

for i in $(seq 1 $NUM_VMS); do
    # Calculate the VMID
    VMID=$((START_VMID + i - 1))

    echo "[$i/$NUM_VMS] Processing VM $VMID..."

    # Start VM, ignore errors
    qm start $VMID || true
done
