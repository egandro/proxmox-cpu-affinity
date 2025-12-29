#!/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPTDIR}/../.env"
fi

WAIT_FOR_AGENT_MAX_LOOP="${WAIT_FOR_AGENT_MAX_LOOP:-100}"
WAIT_FOR_AGENT_SLEEP_SECONDS="${WAIT_FOR_AGENT_SLEEP_SECONDS:-10}"

if [ -z "$1" ]; then
    echo "Usage: $0 <vmid>"
    exit 1
fi
VMID="$1"

echo "Waiting for agent for VM: $VMID"

if ! qm status "$VMID" >/dev/null 2>&1; then
    echo "Warning: VM $VMID does not exist."
    exit 0
fi

if qm config "$VMID" | grep -q "template: 1"; then
    echo "Warning: VM $VMID is a template."
    exit 0
fi

if ! qm status "$VMID" | grep -q "status: running"; then
    echo "VM $VMID is not running."
    exit 1
fi

for _ in $(seq 1 "$WAIT_FOR_AGENT_MAX_LOOP"); do
    if qm guest exec "$VMID" -- /bin/true > /dev/null 2>&1; then
        echo " OK"
        break
    fi
    echo -n "."
    sleep "$WAIT_FOR_AGENT_SLEEP_SECONDS"
done
