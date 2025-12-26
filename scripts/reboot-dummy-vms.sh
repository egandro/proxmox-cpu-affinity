#!/bin/bash

set -e

NUM_VMS=10
START_VMID=200

echo "Rebooting $NUM_VMS dummy VMs starting from ID $START_VMID..."

echo "Warning: This will only reboot VMs created from a real template with a qemu guest agent. The dummy template won't reboot!"

for i in $(seq 1 $NUM_VMS); do
    # Calculate the VMID
    VMID=$((START_VMID + i - 1))

    echo "[$i/$NUM_VMS] Processing VM $VMID..."

    # Reboot VM, ignore errors
    qm reboot $VMID || true
done
