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

TESTCASE_BASE_DIR="${TESTCASE_BASE_DIR:-${SCRIPTDIR}/../testcase}"
TESTCASE="${1:-helloworld}"
TESTCASE_SCRIPT="${2:-init.sh}"

get_ip4() {
    if [ -z "$1" ]; then echo "Usage: get_ip4 <vmid>"; return 1; fi
    qm guest cmd "$1" network-get-interfaces | jq -r '.[]."ip-addresses"[] | select(."ip-address-type"=="ipv4" and ."ip-address" != "127.0.0.1") | ."ip-address"' | head -n 1
}

echo "Running deployed $NUM_VMS testcases ($TESTCASE / $TESTCASE_SCRIPT) on dummy VMs starting from ID $START_VMID..."

for i in $(seq 1 $NUM_VMS); do
    # Calculate the VMID
    VMID=$((START_VMID + i - 1))

    echo "[$i/$NUM_VMS] Processing VM $VMID..."

    echo "detecting first ip4 of VM"
    VM_IP=$(get_ip4 $VMID)

    echo "testing ssh connection ${SSH_USER}@${VM_IP}"
    ssh $SSH_OPTS ${SSH_USER}@${VM_IP} /bin/true || exit 1

    if [ "$NOHUP_MODE" = true ]; then
        echo "Running script ${TESTCASE_SCRIPT} on VM (background)..."
        ssh $SSH_OPTS "${SSH_USER}@${VM_IP}" "sudo chmod +x /testcase/${TESTCASE}/${TESTCASE_SCRIPT}" || exit 1
        ssh $SSH_OPTS "${SSH_USER}@${VM_IP}" "sudo nohup /testcase/${TESTCASE}/${TESTCASE_SCRIPT} $TESTCASE >/dev/null 2>&1 &" || exit 1
    else
        echo "Running script ${TESTCASE_SCRIPT} on VM..."
        ssh $SSH_OPTS "${SSH_USER}@${VM_IP}" "sudo chmod +x /testcase/${TESTCASE}/${TESTCASE_SCRIPT} && sudo /testcase/${TESTCASE}/${TESTCASE_SCRIPT} $TESTCASE" || exit 1
    fi

done
