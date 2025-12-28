#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    . "${SCRIPTDIR}/../.env"
fi

NUM_VMS="${BENCHMARK_NUM_VMS:-2}"
START_VMID="${BENCHMARK_START_VMID:-200}"
SSH_USER="${TESTCASE_SSH_USER:-testcase}"

BASE_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR"
SSH_KEYFILE="${PVE_VM_SSH_KEY_FILE:-"/root/.ssh/id_rsa"}"
SSH_PORT="${TESTCASE_SSH_PORT:-22}"
SSH_OPTS="$BASE_OPTS -p $SSH_PORT -i $SSH_KEYFILE"
SCP_OPTS="$BASE_OPTS -P $SSH_PORT -i $SSH_KEYFILE"

TESTCASE_BASE_DIR="${TESTCASE_BASE_DIR:-${SCRIPTDIR}/../testcase}"
TESTCASE="${1:-helloworld}"
TESTCASE_RESULT_BASE_DIR="${2:-/benchmark/results}"
TESTCASE_RESULT_DIR="${TESTCASE_RESULT_BASE_DIR}/${TESTCASE}"

get_ip4() {
    if [ -z "$1" ]; then echo "Usage: get_ip4 <vmid>"; return 1; fi
    qm guest cmd "$1" network-get-interfaces | jq -r '.[]."ip-addresses"[] | select(."ip-address-type"=="ipv4" and ."ip-address" != "127.0.0.1") | ."ip-address"' | head -n 1
}

echo "Getting results for $NUM_VMS testcases ($TESTCASE) on dummy VMs starting from ID $START_VMID..."

for i in $(seq 1 $NUM_VMS); do
    # Calculate the VMID
    VMID=$((START_VMID + i - 1))

    echo "[$i/$NUM_VMS] Processing VM $VMID..."

    echo "detecting first ip4 of VM"
    VM_IP=$(get_ip4 $VMID)

    echo "testing ssh connection ${SSH_USER}@${VM_IP}"
    ssh $SSH_OPTS ${SSH_USER}@${VM_IP} /bin/true || exit 1

    RESULT_DIR="$TESTCASE_RESULT_DIR/$VMID"
    echo "creating result directory ${RESULT_DIR}"
    mkdir -p ${RESULT_DIR} || exit 1

    echo "fetching results"
    if ssh $SSH_OPTS "${SSH_USER}@${VM_IP}" "[ -d /result/${TESTCASE} ]"; then
        scp -q -r $SCP_OPTS "${SSH_USER}@${VM_IP}:/result/${TESTCASE}" ${RESULT_DIR} || exit 1
    else
        echo "Warning: Remote directory /result/${TESTCASE} does not exist on VM $VMID. Skipping."
    fi
done
