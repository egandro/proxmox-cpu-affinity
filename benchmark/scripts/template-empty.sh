#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPTDIR}/../.env"
fi

# Configuration
VMID="${TEMPLATE_ID_EMPTY:-1001}"

VM_NAME="template-empty"
STORAGE="${PVE_STORAGE:-local-zfs}"
SNIPPET_PATH="${PVE_STORAGE_SNIPPETS_PATH:-/var/lib/vz/snippets}"
CACHE="${CACHE:-writeback}"

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  --overwrite    Delete existing VM with ID $VMID if it exists"
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

if ! pvesm status --storage "$STORAGE" >/dev/null 2>&1; then
    echo "Error: Storage '$STORAGE' does not exist or is not active."
    exit 1
fi

if [ ! -d "$SNIPPET_PATH" ]; then
    echo "Error: Snippet path '$SNIPPET_PATH' does not exist."
    exit 1
fi

echo "Crating template: ${VM_NAME} (${VMID})"

# Check if VM exists
set +e
qm status "$VMID" &>/dev/null
VM_EXISTS=$?
set -e
if [ $VM_EXISTS -eq 0 ]; then
    if [ "$OVERWRITE" != true ]; then
        echo "Error: VM $VMID already exists. Use --overwrite to replace it."
        exit 1
    fi

    echo "Deleting existing VM $VMID..."

    # Check for linked clones
    CLONES=$(grep -l "base-${VMID}-disk" /etc/pve/qemu-server/*.conf 2>/dev/null | grep -v "/${VMID}.conf")
    if [ -n "$CLONES" ]; then
        if [ "$FORCE" = true ]; then
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
            echo "Error: Linked clones found for template $VMID. Use --force to delete them."
            exit 1
        fi
    fi

    if qm status "$VMID" | grep -q "status: running"; then
        qm stop "$VMID" --overrule-shutdown 1 &>/dev/null || true
    fi
    qm destroy "$VMID"
fi

# Create and Configure the VM
echo "Creating VM $VMID ($VM_NAME)..."

# Create the VM with basic settings
qm create "$VMID" --name $VM_NAME --ostype l26

# Memory and CPU settings
qm set "$VMID" --memory 128 --cores 2 --cpu host

# Networking (VirtIO Bridge)
qm set "$VMID" --net0 virtio,bridge=vmbr0

# Create a Dummy Disk
qm set "$VMID" --scsi0 "${STORAGE}:0,size=8G,discard=on,ssd=1,cache=$CACHE"

# We error with trying to boot from the empty disk
qm set "$VMID" --boot order=scsi0

# Finalize
echo "Converting to Template..."
qm template "$VMID"

# Sanity
qm config "$VMID" | grep name

echo "------------------------------------------------"
echo "Done! Template ${VM_NAME} created."
echo "This is a dummy template does nothing when booted."
echo "WARNING: this vm has no qemu agent."
echo "------------------------------------------------"
