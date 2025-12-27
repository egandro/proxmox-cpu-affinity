#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

. ${SCRIPTDIR}/../config.sh

NUM_VMS="${BENCHMARK_NUM_VMS:-2}"
START_VMID="${BENCHMARK_START_VMID:-200}"
SSH_USER="${TESTCASE_SSH_USER:-debian}"

BASE_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR"
SSH_PORT="${TESTCASE_SSH_PORT:-22}"
SSH_OPTS="$BASE_OPTS -p $SSH_PORT"
SCP_OPTS="$BASE_OPTS -P $SSH_PORT"

TESTCASE_BASE_DIR="${TESTCASE_BASE_DIR:-${SCRIPTDIR}/../testcase}"
TESTCASE="${1:-helloworld}"
TESTCASE_ACTION="${2:-install}"
TESTCASE_DIR="${TESTCASE_BASE_DIR}/${TESTCASE}"

if [ ! -d "$TESTCASE_DIR" ]; then
    echo "Error: Testcase directory not found at $TESTCASE_DIR"
    exit 1
fi

get_ip4() {
    if [ -z "$1" ]; then echo "Usage: get_ip4 <vmid>"; return 1; fi
    qm guest cmd "$1" network-get-interfaces | jq -r '.[]."ip-addresses"[] | select(."ip-address-type"=="ipv4" and ."ip-address" != "127.0.0.1") | ."ip-address"' | head -n 1
}

echo "Install or remove $NUM_VMS testcases ($TESTCASE / $TESTCASE_ACTION) on dummy VMs starting from ID $START_VMID..."

for i in $(seq 1 $NUM_VMS); do
    # Calculate the VMID
    VMID=$((START_VMID + i - 1))

    echo "[$i/$NUM_VMS] Processing VM $VMID..."

    echo "detecting first ip4 of VM"
    VM_IP=$(get_ip4 $VMID)

    echo "testing ssh connection ${SSH_USER}@${VM_IP}"
    ssh $SSH_OPTS ${SSH_USER}@${VM_IP} /bin/true || exit 1

    case "$TESTCASE_ACTION" in
        install)
            echo "Installing testcase ${TESTCASE}to VM..."
            ssh $SSH_OPTS "${SSH_USER}@${VM_IP}" "sudo mkdir -p /testcase && sudo chown -R ${SSH_USER} /testcase" || exit 1
            scp -r $SCP_OPTS "$TESTCASE_DIR" "${SSH_USER}@${VM_IP}:/testcase" || exit 1
            ;;
        remove)
            echo "Removing testcases ${TESTCASE} from VM..."
            ssh $SSH_OPTS "${SSH_USER}@${VM_IP}" "sudo rm -rf /testcase" || exit 1
            ;;
        *)
            echo "Error: Unknown testcase action $TESTCASE_ACTION"
            exit 1
            ;;
    esac

done
