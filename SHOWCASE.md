# Showcase

## PS command

```console
root@proxmox:~# proxmox-cpu-affinity ps
VMID     PID        Cores  Sockets  NUMA  Hook-Status  Affinity
----     ---        -----  -------  ----  -----------  --------
5600     13948      4      1        0     Disabled     0-15
20001    6467       2      1        0     Enabled      1,9

root@proxmox:~# proxmox-cpu-affinity hookscript enable 5600

root@proxmox:~# proxmox-cpu-affinity ps
VMID     PID        Cores  Sockets  NUMA  Hook-Status  Affinity
----     ---        -----  -------  ----  -----------  --------
5600     13948      4      1        0     Enabled      0-15
20001    6467       2      1        0     Enabled      1,9

root@proxmox:~# qm stop 5600
root@proxmox:~# qm start 5600

root@proxmox:~# proxmox-cpu-affinity ps
VMID     PID        Cores  Sockets  NUMA  Hook-Status  Affinity
----     ---        -----  -------  ----  -----------  --------
5600     21038      4      1        0     Enabled      1,9,13,14
20001    6467       2      1        0     Enabled      1,9

root@proxmox:~# qm stop 5600
root@proxmox:~# qm start 5600

root@proxmox:~# proxmox-cpu-affinity ps
VMID     PID        Cores  Sockets  NUMA  Hook-Status  Affinity
----     ---        -----  -------  ----  -----------  --------
5600     21538      4      1        0     Enabled      2,4,9,10
20001    6467       2      1        0     Enabled      1,9
```

## Status command

Creates a SVG with heatmap

```console
root@proxmox:~# proxmox-cpu-affinity status svg -o heatmap.svg
root@proxmox:~# proxmox-cpu-affinity status svg --affinity -o heatmap-affinity.svg
```

## Reassign command

(this is usually not required but you can do it)


```console
root@proxmox:~# proxmox-cpu-affinity ps
VMID     PID        Cores  Sockets  NUMA  Hook-Status  Affinity
----     ---        -----  -------  ----  -----------  --------
200      105492     4      1        0     Enabled      2,7,11,12
201      105551     4      1        0     Enabled      4,7,8,12
204      105738     4      1        0     Disabled     0-15

root@proxmox:~# proxmox-cpu-affinity reassign --all --dry-run
VMID     Status     Notes
----     ------     -----
200      dry-run    would reassign
201      dry-run    would reassign
204      skipped    proxmox-cpu-affinity hook not enabled

root@proxmox:~# proxmox-cpu-affinity reassign --all
VMID     Status     Notes
----     ------     -----
200      success
201      success
204      skipped    proxmox-cpu-affinity hook not enabled

root@proxmox:~# proxmox-cpu-affinity ps
VMID     PID        Cores  Sockets  NUMA  Hook-Status  Affinity
----     ---        -----  -------  ----  -----------  --------
200      105492     4      1        0     Enabled      4,10,13,14
201      105551     4      1        0     Enabled      2,9,10,13
204      105738     4      1        0     Disabled     0-15
```
