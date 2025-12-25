#!/bin/bash

if [ ! -d "/etc/pve" ]; then
    echo "Error: This script must be run on a Proxmox VE host (/etc/pve not found)."
    exit 1
fi

SHOW_DETAILS=false
TARGET_VMID=""

usage() {
    echo "Usage: $0 [vmid] [-v|--verbose] [-h]"
    echo "  If no VMID is provided, all running VMs are checked."
    echo "  -v, --verbose   Show detailed thread and child process affinity"
    echo "  -h, --help      Show this help message"
    exit 1
}

while [[ "$#" -gt 0 ]]; do
    case $1 in
        -v|--verbose) SHOW_DETAILS=true ;;
        -h|--help) usage ;;
        *)
            if [[ "$1" =~ ^[0-9]+$ ]]; then
                TARGET_VMID="$1"
            else
                echo "Unknown parameter passed: $1"
                usage
            fi
            ;;
    esac
    shift
done

check_vm() {
    local vmid=$1
    local pid_file="/var/run/qemu-server/$vmid.pid"

    if [ ! -f "$pid_file" ]; then
        if [ -n "$TARGET_VMID" ]; then
            echo "Error: VM $vmid is not running (PID file not found)."
        fi
        return
    fi

    local hook_msg="    "
    if qm config "$vmid" 2>/dev/null | grep -q "hookscript: .*proxmox-cpu-affinity-hook"; then
        hook_msg=" (*)"
    fi

    local pid=$(cat "$pid_file")
    if ps -p "$pid" > /dev/null 2>&1; then
        echo -n "VM $vmid$hook_msg: "
        taskset -cp "$pid"

        if [ "$SHOW_DETAILS" = true ]; then
            echo "  Threads (TID PSR COMMAND):"
            ps -L -p "$pid" -o tid,psr,comm | sed '1d; s/^/    /'

            local children=$(pgrep -P "$pid" | tr '\n' ',' | sed 's/,$//' || true)
            if [ -n "$children" ]; then
                echo "  Child Processes (PID PSR COMMAND):"
                ps -p "$children" -o pid,psr,comm | sed '1d; s/^/    /'
            fi
            echo ""
        fi
    fi
}

if [ -z "$TARGET_VMID" ]; then
    for pid_file in /var/run/qemu-server/*.pid; do
        [ -e "$pid_file" ] || continue
        vmid=$(basename "$pid_file" .pid)
        check_vm "$vmid"
    done
elif [ -n "$TARGET_VMID" ]; then
    check_vm "$TARGET_VMID"
fi
