# proxmox-cpu-affinity

Automated CPU affinity management for Proxmox VE VMs. Uses a background service and a Proxmox hookscript (webhook) to optimize CPU placement on VM startup.

When a VM starts, the hookscript triggers the background service, which then calculates and applies the optimal CPU affinity for that VM.

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

The project includes a CLI tool `proxmox-cpu-affinity` to manage the service and webhooks.

### webhook

Manage hookscript activation. Handles HA and manual affinity checks automatically.
(HA machines are ignored. Templates are ignored. Existing scripts are not overwritten)

```bash
proxmox-cpu-affinity webhook list
proxmox-cpu-affinity webhook enable <VMID> [storage]
proxmox-cpu-affinity webhook enable-all [storage]
proxmox-cpu-affinity webhook disable-all
```

### info

Show current CPU affinity for running VMs.

```bash
proxmox-cpu-affinity info [-v]
proxmox-cpu-affinity info <VMID> [-v]
```

### status

Show current status of the service.

```bash
proxmox-cpu-affinity status
```

### cpuinfo

Runs the cpuinfo and shows the core-to-core latency.

```bash
proxmox-cpu-affinity cpuinfo [-v] [--summary]
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

### Edge case "local" Proxmox Storage is disabled.

The webhook is installed in `/var/lib/vz/snippets/proxmox-cpu-affinity-hook`. This is the default **local** storage.

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

## Components

*   **proxmox-cpu-affinity-service**: Systemd service that monitors VM starts and applies CPU affinity rules (Go HTTP REST Server on `127.0.0.1:8245`).
*   **proxmox-cpu-affinity-hook**: Proxmox hookscript that notifies the service when a VM starts.
*   **proxmox-cpu-affinity**: CLI tool to manage the service, webhooks, view status and CPU topology.

## Algorithm

The algorithm analyzes the host's CPU topology to identify core groups with the lowest inter-core latency.
For every CPU, a vector of neighbors is calculated, ordered from lowest to highest latency. When a VM starts and
requests *n* CPUs (where *n* = cores * sockets), the service assigns a starting CPU and its *n*-1 nearest neighbors.

The starting CPU is selected in a round-robin fashion from the list of all available CPUs to ensure even distribution.

## Files

1.  Binaries are in `/usr/sbin/` and `/usr/bin/`.
2.  Proxmox VM hookscript `/var/lib/vz/snippets/proxmox-cpu-affinity-hook`.
3.  Service `systemctl status proxmox-cpu-affinity.service`
4.  Configuration file `/etc/default/proxmox-cpu-affinity`.

## Resources

- PVE Hook Scripts <https://pve.proxmox.com/pve-docs/pve-admin-guide.html#_hookscripts>
- Original Idea for Core 2 Core Latency <https://github.com/nviennot/core-to-core-latency>

## Testing scripts

The `scripts` folder has some dummy scripts to create, delete, start, and stop VMs.
The hookscript is automatically assigned. This is not installed in the `.deb` package.

## TODO

- The number of CPUs/Sockets etc. might change during the runtime of your host. This is currently not supported (but might be an easy fix).
- `tail -f /var/log/proxmox-cpu-affinity.log` is your UI.

## License

[MIT](LICENSE)
