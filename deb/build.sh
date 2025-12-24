#!/bin/bash
set -e

VERSION="${1:-0.1.0}"
ARCH="${2:-amd64}"

echo "Building Debian package for version ${VERSION} (${ARCH})..."

rm -rf dist
mkdir -p dist/DEBIAN
mkdir -p dist/usr/bin
mkdir -p dist/usr/sbin
mkdir -p dist/var/lib/vz/snippets
mkdir -p dist/etc/systemd/system
mkdir -p dist/etc/default

# Install stripped binaries from build
install -s -m 755 bin/proxmox-cpu-affinity-service dist/usr/sbin/proxmox-cpu-affinity-service
install -s -m 755 bin/proxmox-cpu-affinity-cpuinfo dist/usr/bin/proxmox-cpu-affinity-cpuinfo
install -s -m 755 bin/proxmox-cpu-affinity-hook dist/var/lib/vz/snippets/proxmox-cpu-affinity-hook

# Create systemd service
cp deb/proxmox-cpu-affinity.service dist/etc/systemd/system/

# Create default config file
install -m 644 deb/proxmox-cpu-affinity.default dist/etc/default/proxmox-cpu-affinity

# Create control file
sed -e "s/__VERSION__/${VERSION}/g" -e "s/__ARCH__/${ARCH}/g" deb/control > dist/DEBIAN/control

# Create conffiles to preserve configuration on upgrade
echo "/etc/default/proxmox-cpu-affinity" > dist/DEBIAN/conffiles
chmod 644 dist/DEBIAN/conffiles

# Create postinst script to reload systemd
cp deb/postinst dist/DEBIAN/
chmod 755 dist/DEBIAN/postinst

# Create prerm script to stop service
cp deb/prerm dist/DEBIAN/
chmod 755 dist/DEBIAN/prerm

# Build package
dpkg-deb --root-owner-group --build dist "proxmox-cpu-affinity_${VERSION}_${ARCH}.deb"
rm -rf dist
