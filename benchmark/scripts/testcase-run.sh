#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPTDIR}/../.env"
fi

if [ -z "$1" ]; then
    echo "Usage: $0 <vmid> [testcase] [testcase_script] --nohup"
    exit 1
fi
VMID="$1"

NOHUP_MODE=false
POSITIONAL_ARGS=()

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --nohup) NOHUP_MODE=true ;;
        *) POSITIONAL_ARGS+=("$1") ;;
    esac
    shift
done
set -- "${POSITIONAL_ARGS[@]}"

SSH_USER="${TESTCASE_SSH_USER:-testcase}"

BASE_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR)
SSH_KEYFILE="${PVE_VM_SSH_KEY_FILE:-"/root/.ssh/id_rsa"}"
SSH_PORT="${TESTCASE_SSH_PORT:-22}"
SSH_OPTS=("${BASE_OPTS[@]}" -p "$SSH_PORT" -i "$SSH_KEYFILE")

TESTCASE_BASE_DIR="${TESTCASE_BASE_DIR:-${SCRIPTDIR}/../testcase}"
TESTCASE="${2:-helloworld}"
TESTCASE_SCRIPT="${3:-init.sh}"

get_ip4() {
    if [ -z "$1" ]; then echo "Usage: get_ip4 <vmid>"; return 1; fi
    qm guest cmd "$1" network-get-interfaces | jq -r '.[]."ip-addresses"[] | select(."ip-address-type"=="ipv4" and ."ip-address" != "127.0.0.1") | ."ip-address"' | head -n 1
}

echo "Running deployed testcase ($TESTCASE / $TESTCASE_SCRIPT) on ID: $VMID"

echo "detecting first ip4 of VM"
VM_IP=$(get_ip4 "$VMID")

echo "testing ssh connection ${SSH_USER}@${VM_IP}"
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${VM_IP}" /bin/true || exit 1

CMD_PATH="/testcase/${TESTCASE}/${TESTCASE_SCRIPT}"

if [ "$NOHUP_MODE" = true ]; then
    echo "Running script ${TESTCASE_SCRIPT} on VM (background)..."
    RUN_CMD="sudo nohup $CMD_PATH $TESTCASE >/dev/null 2>&1 &"
else
    echo "Running script ${TESTCASE_SCRIPT} on VM..."
    RUN_CMD="sudo $CMD_PATH $TESTCASE"
fi

REMOTE_CMD="sudo chmod +x $CMD_PATH && $RUN_CMD"
# shellcheck disable=SC2029
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${VM_IP}" "$REMOTE_CMD" || exit 1
