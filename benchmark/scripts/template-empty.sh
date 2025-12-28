#!/bin/bash

# Configuration
VM_ID="${TEMPLATE_ID_EMPTY:-1001}"

VM_NAME="template-empty"
STORAGE="${PVE_STORAGE:-local-zfs}"         # Storage for the VM Disk
CACHE="${CACHE:-writeback}"

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  --overwrite    Delete existing VM with ID $VM_ID if it exists"
    echo "  --force        Force delete linked clones if --overwrite is used"
    echo "  -h, --help     Show this help message"
}

OVERWRITE=false
FORCE=false

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --overwrite) OVERWRITE=true ;;
        --force) FORCE=true ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown parameter: $1"; usage; exit 1 ;;
    esac
    shift
done

echo "Crating template: ${VM_NAME} (${VM_ID})"

# Check if VM exists
if qm status $VM_ID &>/dev/null; then
    if [ "$OVERWRITE" = true ]; then
        echo "Deleting existing VM $VM_ID..."

        # Check for linked clones
        CLONES=$(grep -l "base-${VM_ID}-disk" /etc/pve/qemu-server/*.conf 2>/dev/null | grep -v "/${VM_ID}.conf")
        if [ -n "$CLONES" ]; then
            if [ "$FORCE" = true ]; then
                echo "Found linked clones. Forcing removal..."
                for conf in $CLONES; do
                    CLONE_ID=$(basename "$conf" .conf)
                    echo "Destroying linked clone $CLONE_ID..."
                    qm stop $CLONE_ID --overrule-shutdown 1 &>/dev/null || true
                    qm destroy $CLONE_ID
                done
            else
                echo "Error: Linked clones found for template $VM_ID. Use --force to delete them."
                exit 1
            fi
        fi

        qm stop $VM_ID --overrule-shutdown 1 &>/dev/null || true
        qm destroy $VM_ID
    else
        echo "Error: VM $VM_ID already exists. Use --overwrite to replace it."
        exit 1
    fi
fi

# Create and Configure the VM
echo "Creating VM $VM_ID ($VM_NAME)..."

# Create the VM with basic settings
qm create $VM_ID --name $VM_NAME --ostype l26

# Memory and CPU settings
qm set $VM_ID --memory 128 --cores 2 --cpu host

# Networking (VirtIO Bridge)
qm set $VM_ID --net0 virtio,bridge=vmbr0

# Create a Dummy Disk
qm set $VM_ID --scsi0 ${STORAGE}:0,size=8G,discard=on,ssd=1,cache=$CACHE

# We error with trying to boot from the empty disk
qm set "$VM_ID" --boot order=scsi0

# Finalize
echo "Converting to Template..."
qm template $VM_ID

# Sanity
qm config $VM_ID | grep name

echo "------------------------------------------------"
echo "Done! Template ${VM_NAME} created."
echo "This is a dummy template does nothing when booted."
echo "WARNING: this vm has no qemu agent."
echo "------------------------------------------------"
