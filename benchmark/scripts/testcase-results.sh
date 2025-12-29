#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPTDIR}/../.env"
fi

if [ -z "$1" ]; then
    echo "Usage: $0 <vmid> [testcase] [result_basedir]"
    exit 1
fi
VMID="$1"

SSH_USER="${TESTCASE_SSH_USER:-testcase}"

BASE_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR)
SSH_KEYFILE="${PVE_VM_SSH_KEY_FILE:-"/root/.ssh/id_rsa"}"
SSH_PORT="${TESTCASE_SSH_PORT:-22}"
SSH_OPTS=("${BASE_OPTS[@]}" -p "$SSH_PORT" -i "$SSH_KEYFILE")
SCP_OPTS=("${BASE_OPTS[@]}" -P "$SSH_PORT" -i "$SSH_KEYFILE")

TESTCASE_BASE_DIR="${TESTCASE_BASE_DIR:-${SCRIPTDIR}/../testcase}"
TESTCASE="${2:-helloworld}"
TESTCASE_RESULT_BASE_DIR="${2:-/benchmark/results}"
TESTCASE_RESULT_DIR="${TESTCASE_RESULT_BASE_DIR}/${TESTCASE}"

get_ip4() {
    if [ -z "$1" ]; then echo "Usage: get_ip4 <vmid>"; return 1; fi
    qm guest cmd "$1" network-get-interfaces | jq -r '.[]."ip-addresses"[] | select(."ip-address-type"=="ipv4" and ."ip-address" != "127.0.0.1") | ."ip-address"' | head -n 1
}

echo "Getting results for testcase ($TESTCASE) on VM: $VMID"

echo "detecting first ip4 of VM"
VM_IP=$(get_ip4 "$VMID")

echo "testing ssh connection ${SSH_USER}@${VM_IP}"
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${VM_IP}" /bin/true || exit 1

RESULT_DIR="$TESTCASE_RESULT_DIR/$VMID"
echo "creating result directory ${RESULT_DIR}"
mkdir -p "${RESULT_DIR}" || exit 1

echo "fetching results"
REMOTE_CMD="[ -d /result/${TESTCASE} ]"
# shellcheck disable=SC2029
if ssh "${SSH_OPTS[@]}" "${SSH_USER}@${VM_IP}" "$REMOTE_CMD"; then
    scp -q -r "${SCP_OPTS[@]}" "${SSH_USER}@${VM_IP}:/result/${TESTCASE}" "${RESULT_DIR}" || exit 1
else
    echo "Warning: Remote directory /result/${TESTCASE} does not exist on VM $VMID. Skipping."
fi