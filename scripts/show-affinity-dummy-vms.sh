#!/bin/bash

set -e

#SHOW_DETAILS=false
SHOW_DETAILS=true

if [ -n "$1" ]; then
    VMIDS="$1"
    #SHOW_DETAILS=true
    echo "Checking affinity for VM $1..."
else
    NUM_VMS=10
    START_VMID=200
    echo "Checking affinity for $NUM_VMS dummy VMs starting from ID $START_VMID..."
    VMIDS=$(seq $START_VMID $((START_VMID + NUM_VMS - 1)))
fi

for VMID in $VMIDS; do
    PID_FILE="/var/run/qemu-server/$VMID.pid"
    CONF_FILE="/etc/pve/qemu-server/$VMID.conf"
    HOOK_MSG=""

    if [ -f "$CONF_FILE" ] && ! grep -q "^hookscript:" "$CONF_FILE"; then
        HOOK_MSG=" [NO HOOKSCRIPT]"
    fi

    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p "$PID" > /dev/null 2>&1; then
            echo -n "VM $VMID$HOOK_MSG: "
            taskset -cp "$PID"

            if [ "$SHOW_DETAILS" = true ]; then
                echo "  Threads (TID PSR COMMAND):"
                ps -L -p "$PID" -o tid,psr,comm | sed '1d; s/^/    /'

                CHILDREN=$(pgrep -P "$PID" | tr '\n' ',' | sed 's/,$//' || true)
                if [ -n "$CHILDREN" ]; then
                    echo "  Child Processes (PID PSR COMMAND):"
                    ps -p "$CHILDREN" -o pid,psr,comm | sed '1d; s/^/    /'
                fi
            fi
        else
            echo "VM $VMID$HOOK_MSG: PID file exists ($PID) but process is not running."
        fi
    else
        echo "VM $VMID$HOOK_MSG: Not running."
    fi
done
