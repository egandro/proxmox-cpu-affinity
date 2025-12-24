#!/bin/bash

set -e

#SOURCE_VMID="${1:-902}"
SOURCE_VMID="${1:-1234}"
NUM_VMS=10
CORES=4
MEMORY=1024
START_VMID=200
HOOKSCRIPT="local:snippets/proxmox-cpu-affinity-hook"
AFFINITY="1,3,4,5"

if ! qm status "$SOURCE_VMID" >/dev/null 2>&1; then
    echo "Source VM $SOURCE_VMID does not exist. Creating a new dummy template..."
    qm create "$SOURCE_VMID" --name "dummy-template-$SOURCE_VMID" --memory 128 --net0 virtio,bridge=vmbr0
    qm template "$SOURCE_VMID"
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

    # Configure hook script
    if [ "$i" -ne 5 ]; then
        qm set $NEW_VMID --hookscript $HOOKSCRIPT
    fi

    if [ "$i" -eq 8 ] || [ "$i" -eq 9 ]; then
        qm set $NEW_VMID --affinity $AFFINITY
    fi
done

echo "All $NUM_VMS VMs created and started successfully."
