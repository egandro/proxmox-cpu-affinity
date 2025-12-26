# Showcase

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
