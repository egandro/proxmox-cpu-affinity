#!/bin/bash


if [ ! -d "/testcase" ]; then
    echo "Error: Testcase path '/testcase' does not exist - ensure this only runs in a Testcase Debian VM."
    exit 1
fi

if [ "$(id -u)" -ne 0 ]; then
    echo "Error: This script must be run as root."
    exit 1
fi

echo "Preparing VM for Template use"

set -x
set -e

echo "--- Updating System ---"
apt update && apt dist-upgrade -y
apt autoremove -y
apt clean

# This overrides the default DUID behavior to use the MAC address.
mkdir -p /etc/systemd/network

cat <<EOF > /etc/systemd/network/99-default.link
[Match]
OriginalName=*

[Link]
MACAddressPolicy=persistent
EOF

cat <<EOF > /etc/systemd/network/99-dhcp-mac.network
[Match]
Name=e*

[Network]
DHCP=ipv4

[DHCPv4]
ClientIdentifier=mac
EOF

# If netplan exists, we append the mac identifier to the config.
if command -v netplan > /dev/null; then
    echo "Netplan detected. Applying fix..."
    # We can't easily sed YAML, so we ensure a file exists with the override
    # This might require manual checking if you have complex netplan configs,
    # but for standard templates, this ensures future renders use MAC.
    grep -q "dhcp-identifier: mac" /etc/netplan/*.yaml || echo "WARNING: Please manually add 'dhcp-identifier: mac' to your /etc/netplan/ config if you use Netplan."
fi

# Remove the machine-id file and create an empty one.
# Systemd will generate a new unique ID on the next boot.
rm -f /etc/machine-id
touch /etc/machine-id
rm -f /var/lib/dbus/machine-id
ln -s /etc/machine-id /var/lib/dbus/machine-id
rm -f /var/lib/systemd/random-seed
rm -f /var/lib/systemd/duid
rm -f /var/lib/dhcp/*
rm -f /var/lib/NetworkManager/*.lease

# For Standard Debian (ISC-DHCP-Client / ifupdown).
if [ -f /etc/dhcp/dhclient.conf ]; then
    # Remove old entry if exists to avoid duplicates
    sed -i '/send dhcp-client-identifier/d' /etc/dhcp/dhclient.conf
    echo 'send dhcp-client-identifier = hardware;' >> /etc/dhcp/dhclient.conf
fi

# Clean Cloud-init
if dpkg -l | grep -q cloud-init; then
    echo "Cloud-init detected. Cleaning logs..."
    cloud-init clean --logs --seed
    # Remove generated network configs so they regenerate on next boot
    rm -f /etc/network/interfaces.d/50-cloud-init
    rm -f /etc/netplan/50-cloud-init.yaml
else
    echo "Cloud-init not installed. Skipping."
fi

# Keys will be regenerated on the first boot
rm -f /etc/ssh/ssh_host_*

# Clear audit logs, wtmp, btmp and other log files to reduce image size
truncate -s 0 /var/log/wtmp
truncate -s 0 /var/log/btmp
truncate -s 0 /var/log/lastlog
find /var/log -type f -name "*.log" -exec truncate -s 0 {} \;
find /var/log -type f -name "*.gz" -delete

# Clean history
history -c
unset HISTFILE
rm -f /root/.bash_history
rm -f /home/*/.bash_history

# Delete this script
rm -f "$0"
