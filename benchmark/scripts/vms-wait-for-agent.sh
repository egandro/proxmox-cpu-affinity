#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

. ${SCRIPTDIR}/../config.sh

NUM_VMS="${BENCHMARK_NUM_VMS:-2}"
START_VMID="${BENCHMARK_START_VMID:-200}"

MAX_LOOP=100
SLEEP_TIME=10

echo "Waiting for guest agent to be ready for $NUM_VMS VMs starting from ID $START_VMID..."

for i in $(seq 1 $NUM_VMS); do
    # Calculate the VMID
    VMID=$((START_VMID + i - 1))

    echo "[$i/$NUM_VMS] Processing VM $VMID..."

    # Check if the machine is running
    if ! qm status $VMID | grep -q "status: running"; then
        echo "VM $VMID is not running."
        exit 1
    fi

    # Check if the init process is running
    echo -n "Waiting for agent"
    for j in $(seq 1 $MAX_LOOP); do
        if qm guest exec $VMID -- /bin/true > /dev/null 2>&1; then
            echo " OK"
            break
        fi
        echo -n "."
        sleep $SLEEP_TIME
    done
done
