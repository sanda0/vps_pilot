#!/bin/sh
# Builds the agent's .deb from a built binary.
#
#   agent/packaging/build-deb.sh BINARY TAG ARCH OUT_DIR
#
# TAG is the release tag (v0.2.0, v0.2.0-rc.1), ARCH is amd64 or arm64. The
# package installs the binary to /usr/bin and the same systemd unit as the
# install script. Used by .github/workflows/release.yml.
set -eu
# Package files get 0755/0644 whatever the caller's umask is.
umask 022

[ $# -eq 4 ] || { echo "usage: $0 BINARY TAG ARCH OUT_DIR" >&2; exit 2; }
binary=$1
tag=$2
arch=$3
out=$4
here=$(cd "$(dirname "$0")" && pwd)

# v0.2.0-rc.1 -> 0.2.0~rc.1, so an rc sorts before its release.
version=$(printf '%s' "${tag#v}" | tr '-' '~')

root=$(mktemp -d)
trap 'rm -rf "$root"' EXIT
chmod 0755 "$root"
install -D -m 0755 "$binary" "$root/usr/bin/vps-pilot-agent"
install -d -m 0755 "$root/lib/systemd/system"
sed 's|^ExecStart=/usr/local/bin/|ExecStart=/usr/bin/|' "$here/vps-pilot-agent.service" \
	> "$root/lib/systemd/system/vps-pilot-agent.service"
install -d -m 0755 "$root/DEBIAN"
for script in postinst prerm postrm; do
	install -m 0755 "$here/deb/$script" "$root/DEBIAN/$script"
done
cat > "$root/DEBIAN/control" <<EOF
Package: vps-pilot-agent
Version: $version
Architecture: $arch
Maintainer: Sandakelum <sandakelum.work@gmail.com>
Section: admin
Priority: optional
Homepage: https://vpspilot.dev/
Description: VPS Pilot agent
 Reports this server's CPU, memory, disk and network usage, and the
 projects it hosts, to a VPS Pilot hub over TLS.
EOF

mkdir -p "$out"
dpkg-deb --root-owner-group --build "$root" "$out/vps-pilot-agent_${version}_${arch}.deb"
