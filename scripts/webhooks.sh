#!/bin/bash

# Helper script to enable/disable the proxmox-cpu-affinity hookscript.

if [ ! -d "/etc/pve" ]; then
    echo "Error: This script must be run on a Proxmox VE host (/etc/pve not found)."
    exit 1
fi

# Check if HA manager is available once
HA_MANAGER_BIN=""
if command -v ha-manager >/dev/null 2>&1; then
    HA_MANAGER_BIN="true"
fi
HA_CONFIG_CACHE=""
HA_CONFIG_LOADED=0

usage() {
    echo "Usage: $0 <command> [args...]"
    echo ""
    echo "Commands:"
    echo "  list                      List all VMs and their hook status"
    echo "  status <vmid>             Show hook status for specific VM"
    echo "  enable <vmid> [storage]   Enable hook for specific VM (default storage: local)"
    echo "  disable <vmid>            Disable hook for specific VM"
    echo "  enable-all [storage] [--force] Enable hook for ALL VMs (default storage: local)"
    echo "  disable-all               Disable hook for ALL VMs"
    echo ""
    echo "Note: VMs managed by HA or with manual 'affinity' settings are skipped (bulk) or rejected (single)."
    echo "      Templates are skipped (bulk)."
    echo "      VMs with existing (different) hookscripts are skipped unless --force is used."
    exit 1
}

if [ "$#" -lt 1 ]; then
    usage
fi

ACTION=$1
shift

get_hook_path() {
    local storage="${1:-local}"
    echo "${storage}:snippets/proxmox-cpu-affinity-hook"
}

enable_vm() {
    local vmid=$1
    local storage=$2
    local hook_path=$(get_hook_path "$storage")
    echo "Enabling proxmox-cpu-affinity hook for VM $vmid (storage: $storage)..."
    qm set "$vmid" --hookscript "$hook_path"
}

disable_vm() {
    local vmid=$1
    echo "Disabling proxmox-cpu-affinity hook for VM $vmid..."
    qm set "$vmid" --delete hookscript
}

get_all_vmids() {
    qm list | awk '$1 ~ /^[0-9]+$/ {print $1}'
}

is_ha_vm() {
    local vmid=$1
    if [ -n "$HA_MANAGER_BIN" ]; then
        # Cache HA config on first use
        if [ "$HA_CONFIG_LOADED" -eq 0 ]; then
            HA_CONFIG_CACHE=$(ha-manager config 2>/dev/null || echo "")
            HA_CONFIG_LOADED=1
        fi
        # Check if VM is configured in HA (matches "vm: <vmid>" or "vm:<vmid>")
        if echo "$HA_CONFIG_CACHE" | grep -E -q "^vm:\s*$vmid\b"; then
            return 0
        fi
    fi
    return 1
}

has_affinity_set() {
    local vmid=$1
    if qm config "$vmid" | grep -q "^affinity:"; then
        return 0
    fi
    return 1
}

is_template() {
    local vmid=$1
    if qm config "$vmid" | grep -E -q "^template:\s*1"; then
        return 0
    fi
    return 1
}

print_vm_status_row() {
    local vmid=$1
    local status="DISABLED"
    local notes=""
    local vm_conf=$(qm config "$vmid")

    if echo "$vm_conf" | grep -E -q "^template:\s*1"; then
        notes="VM Template"
    fi

    if is_ha_vm "$vmid"; then
        status="SKIPPED"
        notes="${notes:+$notes, }HA Managed"
    elif echo "$vm_conf" | grep -q "^affinity:"; then
        status="SKIPPED"
        notes="${notes:+$notes, }Manual Affinity Set"
    elif echo "$vm_conf" | grep -q "hookscript: .*proxmox-cpu-affinity-hook"; then
        status="ENABLED"
    fi

    printf "%-8s %-12s %-30s\n" "$vmid" "$status" "$notes"
}

list_vms() {
    printf "%-8s %-12s %-30s\n" "VMID" "HOOK-STATUS" "NOTES"
    printf "%-8s %-12s %-30s\n" "----" "-----------" "-----"

    for vmid in $(get_all_vmids); do
        print_vm_status_row "$vmid"
    done
}

case "$ACTION" in
    list)
        list_vms
        ;;
    status)
        if [ "$#" -lt 1 ]; then usage; fi
        VMID=$1
        if ! qm config "$VMID" >/dev/null 2>&1; then
            echo "Error: VM $VMID not found."
            exit 1
        fi
        printf "%-8s %-12s %-30s\n" "VMID" "HOOK-STATUS" "NOTES"
        printf "%-8s %-12s %-30s\n" "----" "-----------" "-----"
        print_vm_status_row "$VMID"
        ;;
    enable)
        if [ "$#" -lt 1 ]; then usage; fi
        VMID=$1
        STORAGE=${2:-local}
        if is_ha_vm "$VMID"; then
            echo "Error: VM $VMID is managed by HA. Cannot modify hookscript manually."
            exit 1
        fi
        if has_affinity_set "$VMID"; then
            echo "Error: VM $VMID has manual CPU affinity set. Cannot modify hookscript."
            exit 1
        fi
        enable_vm "$VMID" "$STORAGE"
        ;;
    disable)
        if [ "$#" -lt 1 ]; then usage; fi
        VMID=$1
        if is_ha_vm "$VMID"; then
            echo "Error: VM $VMID is managed by HA. Cannot modify hookscript manually."
            exit 1
        fi
        if has_affinity_set "$VMID"; then
            echo "Error: VM $VMID has manual CPU affinity set. Cannot modify hookscript."
            exit 1
        fi
        disable_vm "$VMID"
        ;;
    enable-all)
        STORAGE="local"
        FORCE=0
        while [ "$#" -gt 0 ]; do
            case "$1" in
                --force) FORCE=1 ;;
                *) STORAGE="$1" ;;
            esac
            shift
        done

        for vmid in $(get_all_vmids); do
            if is_ha_vm "$vmid"; then
                echo "Skipping HA-managed VM $vmid..."
                continue
            fi
            vm_conf=$(qm config "$vmid")
            if echo "$vm_conf" | grep -E -q "^template:\s*1"; then
                echo "Skipping VM Template $vmid..."
                continue
            fi
            if echo "$vm_conf" | grep -q "^affinity:"; then
                echo "Skipping VM $vmid (manual affinity set)..."
                continue
            fi
            if echo "$vm_conf" | grep -q "^hookscript:"; then
                if ! echo "$vm_conf" | grep -q "proxmox-cpu-affinity-hook"; then
                    if [ "$FORCE" -eq 0 ]; then
                        echo "Skipping VM $vmid (other hookscript set)..."
                        continue
                    fi
                fi
            fi
            enable_vm "$vmid" "$STORAGE"
        done
        ;;
    disable-all)
        for vmid in $(get_all_vmids); do
            if is_ha_vm "$vmid"; then
                echo "Skipping HA-managed VM $vmid..."
                continue
            fi
            vm_conf=$(qm config "$vmid")
            if echo "$vm_conf" | grep -E -q "^template:\s*1"; then
                echo "Skipping VM Template $vmid..."
                continue
            fi
            if echo "$vm_conf" | grep -q "^affinity:"; then
                echo "Skipping VM $vmid (manual affinity set)..."
                continue
            fi
            if ! echo "$vm_conf" | grep -q "hookscript: .*proxmox-cpu-affinity-hook"; then
                continue
            fi
            disable_vm "$vmid"
        done
        ;;
    *)
        usage
        ;;
esac
