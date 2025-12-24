# proxmox-cpu-affinity

Automated CPU affinity management for Proxmox VE VMs. Uses a background service and hookscript to optimize CPU placement on VM startup.

## Components

*   **proxmox-cpu-affinity-service**: A systemd service that monitors VM starts and applies CPU affinity rules.
*   **proxmox-cpu-affinity-hook**: A Proxmox hookscript that notifies the service when a VM starts.
*   **proxmox-cpu-affinity-cpuinfo**: A utility to gather CPU topology information.

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

### Service Configuration

Edit `/etc/default/proxmox-cpu-affinity` to configure the service:

```bash
# Service Host and Port
PCA_HOST=127.0.0.1
PCA_PORT=8245

# Logging
PCA_LOG_LEVEL=info
PCA_LOG_FILE=/var/log/proxmox-cpu-affinity.log
```

Restart the service after changes:
```bash
systemctl restart proxmox-cpu-affinity
```

### VM Configuration

To enable the affinity management for a specific VM, set the hookscript:

```bash
qm set <VMID> --hookscript local:snippets/proxmox-cpu-affinity-hook
```

## Resources

- PVE Hook Scripts <https://pve.proxmox.com/pve-docs/pve-admin-guide.html#_hookscripts>
- Original Idea for Core 2 Core Latency <https://github.com/nviennot/core-to-core-latency>


## TODO

- The number of CPUs / Sockets etc might change during the runtime of your host. This is currently not supported (but might be an easy fix).

## License
[MIT](LICENSE)
