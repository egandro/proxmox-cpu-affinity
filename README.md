# proxmox-cpu-affinity

Automated CPU affinity management for Proxmox VE VMs. Uses a background service and hookscript to optimize CPU placement on VM startup.

The project is written in Go.

**WARNING**:

- This is alpha code.
- There is absolutely no guarantee that this project will increase the performance. This is an experiment.
- Performance gain is only noticeable on real hardware. You can use a virtual Proxmox server for testing/development. The emulated CPU has random latencies.
- Please test this on multi-socket CPU servers. The socket-to-socket latency is very high, so you can benefit the most.

## Components

*   **proxmox-cpu-affinity-service**: A systemd service that monitors VM starts and applies CPU affinity rules (Go HTTP Rest Server - on `127.0.0.1:8245`)
*   **proxmox-cpu-affinity-hook**: A Proxmox hookscript that notifies the service when a VM starts.
*   **proxmox-cpu-affinity-cpuinfo**: A utility to gather CPU topology information.

## Algorithm

The algorithm analyzes the host's CPU topology to identify core groups with the lowest inter-core latency.
For every CPU, a vector of neighbors is calculated, ordered from lowest to highest latency. When a VM starts and
requests *n* CPUs (where *n* = cores * sockets), the service assigns a starting CPU and its *n*-1 nearest neighbors.
The starting CPU is selected in a round-robin fashion from the list of all available CPUs to ensure even distribution.

## Installation

### Building the Package

You need Go installed to build the project.

```bash
make deb
```

This will create a `.deb` package in the root directory.

### Installing

Copy the `.deb` file to your Proxmox host and install it:

```bash
dpkg -i proxmox-cpu-affinity_*.deb
```

This will:
1.  Install binaries to `/usr/sbin/` and `/usr/bin/`.
2.  Install the hookscript to `/var/lib/vz/snippets/`.
3.  Set up and start the systemd service.
4.  Create a default configuration file at `/etc/default/proxmox-cpu-affinity`.

## Configuration

Service Configuration: `/etc/default/proxmox-cpu-affinity`

Restart the service after changes:
```bash
systemctl restart proxmox-cpu-affinity
```

### VM Configuration

To enable the affinity management for a specific VM, set the hookscript:

**Hint**: If you have no `local` storage, copy the file `/var/lib/vz/snippets/proxmox-cpu-affinity-hook`
to your preferred storage snippets folder.

```bash
qm set <VMID> --hookscript local:snippets/proxmox-cpu-affinity-hook
```

## Resources

- PVE Hook Scripts <https://pve.proxmox.com/pve-docs/pve-admin-guide.html#_hookscripts>
- Original Idea for Core 2 Core Latency <https://github.com/nviennot/core-to-core-latency>

## Testing scripts

The `scripts` folder has some dummy scripts to create, delete, start, and stop VMs.
The hookscript is automatically assigned.

## TODO

- The number of CPUs/Sockets etc. might change during the runtime of your host. This is currently not supported (but might be an easy fix).
- `tail -f /var/log/proxmox-cpu-affinity.log` is your UI.

## License

[MIT](LICENSE)
