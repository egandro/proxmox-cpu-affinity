#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPTDIR}/../.env"
fi

# Configuration
VMID="${TEMPLATE_ID_DEBIAN:-1002}"
OS_TYPE="${OS_TYPE:-debian}"
OS_VERSION="${OS_VERSION:-13}"
OS_RELEASE="${OS_RELEASE:-trixie}"

VM_NAME="template-${VM_NAME:-${OS_TYPE}-${OS_VERSION}-cloud}"
STORAGE="${PVE_STORAGE:-local-zfs}"
SNIPPET_STORAGE="${PVE_STORAGE_SNIPPETS:-local}"
SNIPPET_PATH="${PVE_STORAGE_SNIPPETS_PATH:-/var/lib/vz/snippets}"
SSH_KEYFILE_PUB="${PVE_VM_SSH_KEY_FILE_PUB:-/root/.ssh/id_rsa.pub}"
USERNAME="${DEBIAN_USER:-debian}"
IMAGE_URL="${IMAGE_URL:-https://cloud.debian.org/images/cloud/${OS_RELEASE}/latest/${OS_TYPE}-${OS_VERSION}-genericcloud-amd64.qcow2}"
IMAGE_FILE="${IMAGE_FILE:-${OS_TYPE}-${OS_VERSION}-genericcloud-amd64.qcow2}"
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

if ! pvesm status --storage "$SNIPPET_STORAGE" >/dev/null 2>&1; then
    echo "Error: Snipped Storage '$SNIPPET_STORAGE' does not exist or is not active."
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

# Download the Image
echo "Downloading Image..."
wget -q -c -O "$IMAGE_FILE" "$IMAGE_URL"

# Create and Configure the VM
echo "Creating VM $VMID ($VM_NAME)..."

# Create the VM with basic settings
qm create "$VMID" --name "$VM_NAME" --ostype l26

# Memory and CPU settings
qm set "$VMID" --memory 1024 --cores 2 --cpu host

# Networking (VirtIO Bridge)
qm set "$VMID" --net0 virtio,bridge=vmbr0

# Serial Console Settings (Required for Cloud-Init debugging and Proxmox Console)
qm set "$VMID" --serial0 socket --vga serial0

# Import the Disk
echo "Importing disk to $PVE_STORAGE..."
qm set "$VMID" --scsi0 "${STORAGE}:0,import-from=$(pwd)/$IMAGE_FILE,discard=on,ssd=1,cache=$CACHE" 1> /dev/null
qm set "$VMID" --boot order=scsi0 --scsihw virtio-scsi-single

# Create the a custom Cloud-Init Snippet for QEMU Agent
# This file tells cloud-init to install the agent on first boot
echo "Creating Cloud-Init snippet for QEMU Guest Agent..."

if [ ! -f "$SSH_KEYFILE_PUB" ]; then
    echo "Error: SSH Keyfile not found at $SSH_KEYFILE_PUB"
    exit 1
fi
SSH_KEY=$(cat "$SSH_KEYFILE_PUB")

mkdir -p "$SNIPPET_PATH"
cat << EOF > "$SNIPPET_PATH/${OS_TYPE}-${OS_VERSION}-agent.yaml"
#cloud-config
users:
  - default
  - name: $USERNAME
    ssh_authorized_keys:
      - $SSH_KEY
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: sudo
    shell: /bin/bash
package_update: true
package_upgrade: true
packages:
  - qemu-guest-agent
runcmd:
  - systemctl start qemu-guest-agent
  - systemctl enable qemu-guest-agent
EOF

# Configure Cloud-Init Drive
qm set "$VMID" --ide2 "${STORAGE}:cloudinit"

# Apply Cloud-Init Settings & Attach Snippet
echo "Applying Cloud-Init configuration..."
echo "WARNING: this disables (most) of the Proxmox Cloud Init UI support"

# Attach the User Data Snippet
# Syntax: user=<storage>:snippets/<filename>
qm set "$VMID" --cicustom "user=${SNIPPET_STORAGE}:snippets/${OS_TYPE}-${OS_VERSION}-agent.yaml"

# Set the cloud init user + keyfile (might not work)
qm set "$VMID" --ciuser "$USERNAME"
qm set "$VMID" --sshkeys "$SSH_KEYFILE_PUB"

# Set IP Configuration (IPv6 auto, IPv4 DHCP)
qm set "$VMID" --ipconfig0 "ip6=auto,ip=dhcp"

# I have no idea how to use this when we user our own custom snippet
# # SSH Keys and User
# qm set "$VMID" --sshkeys $SSH_KEYFILE_PUB
# qm set "$VMID" --ciuser $USERNAME

# Enable Agent Flag in Proxmox (So Proxmox knows to look for it)
qm set "$VMID" --agent enabled=1,fstrim_cloned_disks=1

# Resize Disk (Optional: expand to 8GB min)
qm disk resize "$VMID" scsi0 8G

# Finalize
echo "Converting to Template..."
qm template "$VMID"

# Sanity
qm config "$VMID" | grep name

echo "------------------------------------------------"
echo "Done! Template ${OS_TYPE}-${OS_VERSION} ($VMID) created."
echo "When you clone this, wait for the first boot to finish."
echo "Cloud-init will install qemu-guest-agent automatically."
echo "------------------------------------------------"
