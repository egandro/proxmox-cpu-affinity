# Proxmox Patch

This is dangerous!

**WARNING**: Do not do this on a production server.

The Patch will help `proxmox-cpu-affinity` to run without hookscripts.

**Install patch command**

```bash
apt-get install patch
```

**Download the Patch**

```bash
rm -f /tmp/feature.diff
wget https://github.com/egandro/qemu-server/pull/1.diff -O /tmp/feature.diff
head -n 5 /tmp/feature.diff
# You will likely see a line like --- a/PVE/QemuServer.pm or --- a/src/PVE/QemuServer.pm.
```

**Backup the current QemuServer.pm**

```bash
cd /usr/share/perl5

# Create a backup with a timestamp to ensure uniqueness
cp PVE/QemuServer.pm PVE/QemuServer.pm.backup-$(date +%s)

# Verify the backup exists
ls -l PVE/QemuServer.pm.backup*
```

**Test if the patch can be applied**

```bash
cd /usr/share/perl5
patch -p2 --dry-run < /tmp/feature.diff
```

**Apply the patch**

```bash
systemctl stop pvedaemon pveproxy
patch -p2 < /tmp/feature.diff
systemctl start pvedaemon pveproxy
```

**View Logs*

```bash
journalctl -f
journalctl -u pvedaemon -f
```

**Undo the Patch**

```bash
cd /usr/share/perl5
systemctl stop pvedaemon pveproxy
patch -R -p2 < /tmp/feature.diff
systemctl start pvedaemon pveproxy
```

**Optional - Restore Backup**

```bash
systemctl stop pvedaemon pveproxy

cd /usr/share/perl5

# Locate your backup file
ls -l PVE/QemuServer.pm.backup-*

# Restore it (replace timestamp with your actual one)
cp PVE/QemuServer.pm.backup-XXXXXXX PVE/QemuServer.pm

# delete force CPUAffinityServiceClient.pm
rm -f PVE/QemuServer/CPUAffinityServiceClient.pm

systemctl start pvedaemon pveproxy
```
