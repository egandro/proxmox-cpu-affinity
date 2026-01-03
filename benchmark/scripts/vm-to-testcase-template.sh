#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPTDIR}/../.env"
fi

WAIT_FOR_AGENT_MAX_LOOP="${WAIT_FOR_AGENT_MAX_LOOP:-100}"
WAIT_FOR_AGENT_SLEEP_SECONDS="${WAIT_FOR_AGENT_SLEEP_SECONDS:-10}"

SSH_USER="${TESTCASE_SSH_USER:-debian}"

BASE_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR)
SSH_KEYFILE="${PVE_VM_SSH_KEY_FILE:-"/root/.ssh/id_rsa"}"
SSH_PORT="${TESTCASE_SSH_PORT:-22}"
SSH_OPTS=("${BASE_OPTS[@]}" -p "$SSH_PORT" -i "$SSH_KEYFILE")
SCP_OPTS=("${BASE_OPTS[@]}" -P "$SSH_PORT" -i "$SSH_KEYFILE")

usage() {
    echo "Usage: $0 <vmid> <testcase> <name_postfix> <init_script> <cleanup_script>"
    exit 1
}

if [ "$#" -lt 5 ]; then
    usage
fi

VMID="$1"
TESTCASE="$2"
NEW_NAME_POSTFIX="$3"
TESTCASE_SCRIPT="$4"
CLEANUP_SCRIPT="$5"

TESTCASE_BASE_DIR="${TESTCASE_BASE_DIR:-${SCRIPTDIR}/../testcase}"
TESTCASE_DIR="${TESTCASE_BASE_DIR}"

if [ ! -d "$TESTCASE_DIR" ]; then
    echo "Error: Testcase directory not found at $TESTCASE_DIR"
    exit 1
fi

if [ ! -f "$TESTCASE_DIR/$TESTCASE/$TESTCASE_SCRIPT" ]; then
    echo "Error: Init script $TESTCASE_DIR/$TESTCASE/$TESTCASE_SCRIPT does not exist."
    exit 1
fi

if [ ! -f "$CLEANUP_SCRIPT" ]; then
    echo "Error: Cleanup script $CLEANUP_SCRIPT does not exist."
    exit 1
fi

echo "Creating new template $NEW_VMID from VM: $VMID (testcase: $TESTCASE, postfix: $NEW_NAME_POSTFIX) with script $TESTCASE_FOLDER"

if ! qm status "$VMID" >/dev/null 2>&1; then
    echo "Error: VM $VMID does not exist."
    exit 1
fi

if ! qm config "$VMID" | grep -qE "agent: (1|enabled=1)"; then
    echo "Error: VM $VMID does not have the QEMU guest agent enabled."
    exit 1
fi

if ! qm status "$VMID" | grep -q "status: running"; then
    echo "Error: VM $VMID is not running."
    exit 1
fi

if qm config "$VMID" | grep -q "template: 1"; then
    echo "Error: VM $VMID is already a template."
    exit 1
fi

echo "Waiting for agent for VM: $VMID"

for _ in $(seq 1 "$WAIT_FOR_AGENT_MAX_LOOP"); do
    if qm guest exec "$VMID" -- /bin/true > /dev/null 2>&1; then
        echo " OK"
        break
    fi
    echo -n "."
    sleep "$WAIT_FOR_AGENT_SLEEP_SECONDS"
done

get_ip4() {
    if [ -z "$1" ]; then echo "Usage: get_ip4 <vmid>"; return 1; fi
    qm guest cmd "$1" network-get-interfaces | jq -r '.[]."ip-addresses"[] | select(."ip-address-type"=="ipv4" and ."ip-address" != "127.0.0.1") | ."ip-address"' | head -n 1
}

echo "detecting first ip4 of VM"
VM_IP=$(get_ip4 "$VMID")

echo "testing ssh connection ${SSH_USER}@${VM_IP}"
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${VM_IP}" /bin/true || exit 1

echo "Uploading script"
REMOTE_CMD="sudo mkdir -p /testcase && sudo chown -R ${SSH_USER} /testcase"
# shellcheck disable=SC2029
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${VM_IP}" "$REMOTE_CMD" || exit 1
scp -r "${SCP_OPTS[@]}" "$TESTCASE_DIR/${TESTCASE}" "${SSH_USER}@${VM_IP}:/testcase" || exit 1

CMD_PATH="/testcase/${TESTCASE}"
RUN_CMD="sudo $CMD_PATH/$TESTCASE_SCRIPT $TESTCASE"

echo TESTCASE_DIR: "$TESTCASE_DIR"
echo TESTCASE: "$TESTCASE"
echo CMD_PATH: "$CMD_PATH"
echo RUN_CMD: "$RUN_CMD"

echo "Running script ${TESTCASE_SCRIPT} on VM..."

REMOTE_CMD="sudo chmod +x $CMD_PATH && $RUN_CMD"
# shellcheck disable=SC2029
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${VM_IP}" "$REMOTE_CMD" || exit 1

CLEAN_CMD_PATH="/tmp/cleanup.sh"

echo "Uploading cleanup script ${CLEANUP_SCRIPT} on VM..."
scp "${SCP_OPTS[@]}" "$CLEANUP_SCRIPT" "${SSH_USER}@${VM_IP}:${CLEAN_CMD_PATH}" || exit 1

echo "Running script ${CLEANUP_SCRIPT} on VM..."

RUN_CMD="sudo $CLEAN_CMD_PATH"
REMOTE_CMD="sudo chmod +x $CMD_PATH && $RUN_CMD"

# shellcheck disable=SC2029
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${VM_IP}" "$REMOTE_CMD" || exit 1

echo "Stopping VM..."

# Stopping VM
qm stop "$VMID"

echo "Creating Template..."

CURRENT_NAME=$(qm config "$VMID" | grep "name: " | awk '{print $2}')
# remove "-$VMID" or "$VMID-"
BASE_NAME="${CURRENT_NAME%-"$VMID"}"
BASE_NAME="${BASE_NAME%"$VMID"-}"
NEW_NAME="template-${BASE_NAME}-${NEW_NAME_POSTFIX}"

echo "Renaming Template to $NEW_NAME ..."
qm set "$VMID" --name "$NEW_NAME"

# Convert to template
qm template "$VMID"
