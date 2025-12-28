#!/bin/bash

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    . "${SCRIPTDIR}/../.env"
fi

# Configuration
VM_ID="${TEMPLATE_ID_DEBIAN:-1002}"
OS_TYPE="${OS_TYPE:-debian}"
OS_VERSION="${OS_VERSION:-13}"
OS_RELEASE="${OS_RELEASE:-trixie}"

VM_NAME="template-${VM_NAME:-${OS_TYPE}-${OS_VERSION}-cloud}"
STORAGE="${PVE_STORAGE:-local-zfs}"
SNIPPET_STORAGE="${PVE_STORAGE_SNIPPETS:-local}"
SNIPPET_PATH="${SNIPPET_PATH:-/var/lib/vz/snippets}"
SSH_KEYFILE_PUB="${PVE_VM_SSH_KEY_FILE_PUB:-/root/.ssh/id_rsa.pub}"
USERNAME="${DEBIAN_USER:-debian}"
IMAGE_URL="${IMAGE_URL:-https://cloud.debian.org/images/cloud/${OS_RELEASE}/latest/${OS_TYPE}-${OS_VERSION}-genericcloud-amd64.qcow2}"
IMAGE_FILE="${IMAGE_FILE:-${OS_TYPE}-${OS_VERSION}-genericcloud-amd64.qcow2}"
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

# Create the Cloud-Init Snippet for QEMU Agent
# This file tells cloud-init to install the agent on first boot
echo "Creating Cloud-Init snippet for QEMU Guest Agent..."

if [ ! -f "$SSH_KEYFILE_PUB" ]; then
    echo "Error: SSH Keyfile not found at $SSH_KEYFILE_PUB"
    exit 1
fi
SSH_KEY=$(cat "$SSH_KEYFILE_PUB")

mkdir -p $SNIPPET_PATH
cat << EOF > $SNIPPET_PATH/${OS_TYPE}-${OS_VERSION}-agent.yaml
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

# Download the Image
echo "Downloading Image..."
wget -q -c -O "$IMAGE_FILE" "$IMAGE_URL"

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
qm set $VM_ID --memory 1024 --cores 2 --cpu host

# Networking (VirtIO Bridge)
qm set $VM_ID --net0 virtio,bridge=vmbr0

# Serial Console Settings (Required for Cloud-Init debugging and Proxmox Console)
qm set $VM_ID --serial0 socket --vga serial0

# Import the Disk
echo "Importing disk to $PVE_STORAGE..."
qm set $VM_ID --scsi0 ${STORAGE}:0,import-from="$(pwd)/$IMAGE_FILE",discard=on,ssd=1,cache=$CACHE 1> /dev/null
qm set $VM_ID --boot order=scsi0 --scsihw virtio-scsi-single

# Configure Cloud-Init Drive
qm set $VM_ID --ide2 ${STORAGE}:cloudinit

# Apply Cloud-Init Settings & Attach Snippet
echo "Applying Cloud-Init configuration..."

# Attach the User Data Snippet
# Syntax: user=<storage>:snippets/<filename>
qm set $VM_ID --cicustom "user=${SNIPPET_STORAGE}:snippets/${OS_TYPE}-${OS_VERSION}-agent.yaml"

# Set IP Configuration (IPv6 auto, IPv4 DHCP)
qm set $VM_ID --ipconfig0 "ip6=auto,ip=dhcp"

# I have no idea how to use this when we user our own custom snippet
# # SSH Keys and User
# qm set $VM_ID --sshkeys $SSH_KEYFILE_PUB
# qm set $VM_ID --ciuser $USERNAME

# Enable Agent Flag in Proxmox (So Proxmox knows to look for it)
qm set $VM_ID --agent enabled=1,fstrim_cloned_disks=1

# Resize Disk (Optional: expand to 8GB min)
qm disk resize $VM_ID scsi0 8G

# Finalize
echo "Converting to Template..."
qm template $VM_ID

# Sanity
qm config $VM_ID | grep name

echo "------------------------------------------------"
echo "Done! Template ${OS_TYPE}-${OS_VERSION} ($VM_ID) created."
echo "When you clone this, wait for the first boot to finish."
echo "Cloud-init will install qemu-guest-agent automatically."
echo "------------------------------------------------"
