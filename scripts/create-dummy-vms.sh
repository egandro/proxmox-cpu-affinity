#!/bin/bash

set -e

TEMPLATE_NAME="dummy-pve-affinity-template"

if [ -n "$1" ]; then
    SOURCE_VMID="$1"
else
    EXISTING_ID=$(qm list | awk -v name="$TEMPLATE_NAME" '$2 == name {print $1}' | head -n 1)

    if [ -n "$EXISTING_ID" ]; then
        echo "Found existing template '$TEMPLATE_NAME' with VMID $EXISTING_ID."
        SOURCE_VMID="$EXISTING_ID"
    else
        SOURCE_VMID=$(pvesh get /cluster/nextid)
    fi
fi

NUM_VMS=10
CORES=4
MEMORY=1024
START_VMID=200
HOOKSCRIPT="local:snippets/proxmox-cpu-affinity-hook"
AFFINITY="1,3,4,5"

if ! qm status "$SOURCE_VMID" >/dev/null 2>&1; then
    echo "Source VM $SOURCE_VMID does not exist. Creating a new dummy template '$TEMPLATE_NAME'..."
    qm create "$SOURCE_VMID" --name "$TEMPLATE_NAME" --memory 128 --net0 virtio,bridge=vmbr0
    # we disable the booting - the default net0 boot will just eat CPU time
    qm set "$SOURCE_VMID" --delete boot
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
