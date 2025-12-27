# proxmox-cpu-affinity

Automated CPU affinity management for Proxmox VE VMs. Uses a background service and a Proxmox hookscript to optimize CPU placement on VM startup.

When a VM starts, the hookscript triggers the background service, which then calculates and applies the optimal CPU affinity for that VM.

See [SHOWCASE.md](SHOWCASE.md) for an example session and [example.svg](screenshots/example.svg) for a SVG heatmap.

Written in Go.

**WARNING**:

- This is alpha code.
- There is no guarantee that this project will increase performance. This is an experiment.
- Performance gains are only noticeable on bare-metal hardware. Virtual environments have random CPU latencies.
- Best results are expected on multi-socket servers where socket-to-socket latency is significant.

## Installation

### Release

Download the latest `.deb` package from Releases.

### Source

Requires Go.

```bash
make deb
dpkg -i proxmox-cpu-affinity_*.deb
```

## CLI Tool

The project includes a CLI tool `proxmox-cpu-affinity` to manage the service and hookscripts.

### hookscript

Manage hookscript activation. Handles HA and manual affinity checks automatically.
(HA machines are ignored. Templates are ignored. Existing scripts are not overwritten.)

```bash
proxmox-cpu-affinity hookscript list [--json] [--quiet]
proxmox-cpu-affinity hookscript enable <VMID> [storage] [--dry-run]
proxmox-cpu-affinity hookscript disable [--dry-run]
proxmox-cpu-affinity hookscript enable-all [storage] [--dry-run]
proxmox-cpu-affinity hookscript disable-all [--dry-run]
```

### ps

Show current CPU affinity for running VMs.

```bash
proxmox-cpu-affinity ps [-v]
proxmox-cpu-affinity ps <VMID> [-v] [--json] [--quiet]
```

### status

Show current status of the service.

```bash
proxmox-cpu-affinity status
proxmox-cpu-affinity status ping [--json]
proxmox-cpu-affinity status core-ranking [--json]
proxmox-cpu-affinity status core-ranking-summary [--json]
proxmox-cpu-affinity status core-vm-affinity [--json]
proxmox-cpu-affinity status svg [--affinity]
```

### cpuinfo

Runs the cpuinfo and shows the core-to-core latency.

```bash
proxmox-cpu-affinity cpuinfo [-v] [--summary] [--quiet]
```

### reassign

Reassign CPU affinity for running VMs with enabled hooks.

```bash
proxmox-cpu-affinity reassign [vmid] [--all] [--dry-run]
```

## Manual VM Configuration

To enable the affinity management for a specific VM, set the hookscript:

```bash
# "local" is a Proxmox Storage ID
qm set <VMID> --hookscript local:snippets/proxmox-cpu-affinity-hook
```

Disable the hookscript

```bash
qm set <VMID> --delete hookscript
```

**Warning**: Your machine will fail to start, in case the hookscript does not exist!

- you removed proxmox-cpu-affinity and didn't change the configuration
- you have a cluster and don't have proxmox-cpu-affinity installed on every node

### Edge case "local" Proxmox Storage is disabled

The hookscript is installed in `/var/lib/vz/snippets/proxmox-cpu-affinity-hook`. This is the default **local** storage.

In case you disabled your **local** storage you have to link it to a custom storage.

```bash
cat /etc/pve/storage.cfg  | grep raid
dir: raid
        path /raid
mkdir -p /raid/snippets/
ln -s /var/lib/vz/snippets/proxmox-cpu-affinity-hook /raid/snippets/proxmox-cpu-affinity-hook

# you can now use it from "raid"
# qm set <VMID> --hookscript raid:snippets/proxmox-cpu-affinity-hook
```

**Info** This might be a bug in Proxmox. Regardless the status of "local" you can enable / use a hookscript (even if the storage disabled).

## Components

*   **proxmox-cpu-affinity-service**: Systemd service that monitors VM starts and applies CPU affinity rules (Go HTTP REST Server on `127.0.0.1:8245`).
*   **proxmox-cpu-affinity-hook**: Proxmox hookscript that notifies the service when a VM starts.
*   **proxmox-cpu-affinity**: CLI tool to manage the service, hookscript, view status and CPU topology.

## Algorithm

The algorithm analyzes the host's CPU topology to identify core groups with the lowest inter-core latency.
For every CPU, a vector of neighbors is calculated, ordered from lowest to highest latency. When a VM starts and
requests *n* CPUs (where *n* = cores * sockets), the service assigns a starting CPU and its *n*-1 nearest neighbors.

The starting CPU is selected in a round-robin fashion from the list of all available CPUs to ensure even distribution.

## CPU Hotplug Watchdog

The service monitors CPU hotplug events. When CPUs are added or removed, it automatically recalculates the core-to-core latency matrix.

This ensures that the affinity logic always uses the current CPU topology without requiring a service restart.

To test this (e.g. nested Proxmox):

```bash
qm set <VMID> --numa 1 --hotplug disk,network,usb,memory,cpu # numa + cpu hotplug must be enabled
qm set <VMID> --sockets 4 --cores 4 --vcpus 4 # start with 4 CPUs

# start the nested proxmox with proxmox-cpu-affinity
# open the logfile

# Add or remove CPUs at runtime (here increment)
qm set <VMID> -vcpus 6
```

## Files

1.  Proxmox VM hookscript `/var/lib/vz/snippets/proxmox-cpu-affinity-hook`.
2.  Configuration file `/etc/default/proxmox-cpu-affinity`.

## Resources

- PVE Hook Scripts <https://pve.proxmox.com/pve-docs/pve-admin-guide.html#_hookscripts>
- Original Idea for Core 2 Core Latency <https://github.com/nviennot/core-to-core-latency>

## Testing scripts

The `scripts` folder has some dummy scripts to create, delete, start, and stop VMs.
The hookscript is automatically assigned. This is not installed in the `.deb` package.

## TODO

- Current UI `tail -f /var/log/proxmox-cpu-affinity.log`
- Recalculate `AdaptiveCpuInfoParameters` after a CPU hotplug event.
- Deal with `GetSelections` when VMs are stopped. Since we are not a realtime information - just a affinity selector, this might be accurate.
- Try to get a [Proxmox Patch](PROXMOX-PATCH.md)  applied. This is a draft version: <https://github.com/egandro/qemu-server/pull/1>. It will help us get rid of hookscripts.

## License

[MIT](LICENSE)
