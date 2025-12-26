#!/bin/bash
set -e

VERSION="${1:-0.0.1}"
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
install -s -m 755 bin/$(ARCH)/proxmox-cpu-affinity-service dist/usr/sbin/proxmox-cpu-affinity-service
install -s -m 755 bin/$(ARCH)/proxmox-cpu-affinity dist/usr/bin/proxmox-cpu-affinity
install -s -m 755 bin/$(ARCH)/proxmox-cpu-affinity-hook dist/var/lib/vz/snippets/proxmox-cpu-affinity-hook

if [ "$GITHUB_ACTIONS" != "true" ]; then
    echo "Developer build detected: Installing testing scripts..."
    mkdir -p dist/usr/share/proxmox-cpu-affinity
    install -m 755 scripts/create-dummy-vms.sh dist/usr/share/proxmox-cpu-affinity
    install -m 755 scripts/destroy-dummy-vms.sh dist/usr/share/proxmox-cpu-affinity
    install -m 755 scripts/start-dummy-vms.sh dist/usr/share/proxmox-cpu-affinity
    install -m 755 scripts/stop-dummy-vms.sh dist/usr/share/proxmox-cpu-affinity
else
    echo "GitHub Actions detected: Skipping installation of testing scripts."
fi

# Create systemd service
install -m 644 deb/proxmox-cpu-affinity.service dist/etc/systemd/system/

# Create default config file
install -m 644 deb/proxmox-cpu-affinity.default dist/etc/default/proxmox-cpu-affinity

# Create control file
sed -e "s/__VERSION__/${VERSION}/g" -e "s/__ARCH__/${ARCH}/g" deb/control > dist/DEBIAN/control

# Create conffiles to preserve configuration on upgrade
install -m 644 deb/conffiles dist/DEBIAN/

# Create postinst script to reload systemd
install -m 755 deb/postinst dist/DEBIAN/

# Create prerm script to stop service
install -m 755 deb/prerm dist/DEBIAN/

# Create postrm script to remove the completion files
install -m 755 deb/postrm dist/DEBIAN/

# Build package
dpkg-deb --root-owner-group --build dist "proxmox-cpu-affinity_${VERSION}_${ARCH}.deb"
rm -rf dist
