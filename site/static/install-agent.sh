#!/bin/sh
# Installs or upgrades the VPS Pilot agent as a systemd service.
#
#   curl -fsSL https://vpspilot.dev/install-agent.sh \
#     | sudo VPS_PILOT_TOKEN=vpsp1... sh -s -- --hub hub.example.com:55001
#
# Options:
#   --hub HOST[:PORT]    Hub address. Required on the first install.
#   --version vX.Y.Z     Release to install. Default: the latest release.
#   --projects-dir DIRS  Comma-separated directories to search for
#                        config.vpspilot.json. Default: /var/www,/home.
#   --token TOKEN        Join token. Prefer VPS_PILOT_TOKEN so it stays out
#                        of ps. Only needed until the agent has enrolled.
#
# Running it again upgrades the agent and keeps its enrollment. Options given
# replace the saved ones; options left out keep their saved value.
set -eu

RELEASES="${VPS_PILOT_RELEASES:-https://github.com/sanda0/vps_pilot/releases}"
BIN=/usr/local/bin/vps-pilot-agent
UNIT=/etc/systemd/system/vps-pilot-agent.service
CONF_DIR=/etc/vps-pilot-agent
ENV_FILE=$CONF_DIR/agent.env
STATE=/var/lib/vps-pilot-agent/state.json

say() { printf '%s\n' "$*"; }
die() { printf 'install-agent: %s\n' "$*" >&2; exit 1; }

# set_env KEY [VALUE] replaces KEY's line in the env file, or removes it.
set_env() {
	grep -v "^$1=" "$ENV_FILE" > "$ENV_FILE.new" 2>/dev/null || true
	if [ $# -gt 1 ]; then
		printf '%s=%s\n' "$1" "$2" >> "$ENV_FILE.new"
	fi
	mv "$ENV_FILE.new" "$ENV_FILE"
}

main() {
	hub=""
	version=""
	projects_dirs=""
	token="${VPS_PILOT_TOKEN:-}"
	while [ $# -gt 0 ]; do
		case "$1" in
			--hub | --version | --projects-dir | --token)
				[ $# -gt 1 ] || die "$1 needs a value"
				case "$1" in
					--hub) hub=$2 ;;
					--version) version=$2 ;;
					--projects-dir) projects_dirs=$2 ;;
					--token) token=$2 ;;
				esac
				shift 2
				;;
			-h | --help)
				sed -n '2,16s/^# \{0,1\}//p' "$0" 2>/dev/null || true
				exit 0
				;;
			*) die "unknown option: $1" ;;
		esac
	done

	[ "$(id -u)" = 0 ] || die "run it as root, for example with sudo"
	[ "$(uname -s)" = Linux ] || die "the agent runs on Linux only"
	for cmd in curl tar sha256sum systemctl; do
		command -v "$cmd" > /dev/null || die "$cmd is required"
	done
	case "$(uname -m)" in
		x86_64 | amd64) arch=amd64 ;;
		aarch64 | arm64) arch=arm64 ;;
		*) die "no agent build for $(uname -m); builds exist for amd64 and arm64" ;;
	esac

	[ -n "$hub" ] || [ -f "$ENV_FILE" ] || die "--hub is required on the first install"
	enrolled=false
	[ -s "$STATE" ] && enrolled=true
	if [ "$enrolled" = false ] && [ -z "$token" ]; then
		die "the agent isn't enrolled yet: pass the join token from Nodes → Add node in VPS_PILOT_TOKEN"
	fi

	if [ -z "$version" ]; then
		latest=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$RELEASES/latest") ||
			die "couldn't reach $RELEASES"
		version=${latest##*/}
	fi
	case "$version" in
		v*) ;;
		*) die "couldn't find the latest release at $RELEASES" ;;
	esac

	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT
	archive="vps-pilot-agent-$version-linux-$arch.tar.gz"
	say "Downloading $archive"
	curl -fsSL -o "$tmp/$archive" "$RELEASES/download/$version/$archive" ||
		die "couldn't download $version for $arch"
	curl -fsSL -o "$tmp/SHA256SUMS" "$RELEASES/download/$version/SHA256SUMS" ||
		die "couldn't download SHA256SUMS for $version"
	(cd "$tmp" && grep " $archive\$" SHA256SUMS | sha256sum -c - > /dev/null 2>&1) ||
		die "$archive doesn't match SHA256SUMS"
	tar -xzf "$tmp/$archive" -C "$tmp"
	[ -f "$tmp/vps-pilot-agent.service" ] ||
		die "$version has no systemd unit; this script installs v0.2.0 or later"

	install -m 0755 "$tmp/vps-pilot-agent" "$BIN"
	install -m 0644 "$tmp/vps-pilot-agent.service" "$UNIT"
	install -d -m 0700 "$CONF_DIR"
	umask 077
	touch "$ENV_FILE"
	[ -z "$hub" ] || set_env VPS_PILOT_HUB "$hub"
	[ -z "$projects_dirs" ] || set_env VPS_PILOT_PROJECT_DIRS "$projects_dirs"
	if [ "$enrolled" = false ]; then
		set_env VPS_PILOT_TOKEN "$token"
	fi

	systemctl daemon-reload
	systemctl enable vps-pilot-agent > /dev/null 2>&1
	systemctl restart vps-pilot-agent

	if [ "$enrolled" = false ]; then
		say "Waiting for the agent to enroll"
		i=0
		while [ ! -s "$STATE" ] && [ $i -lt 30 ]; do
			sleep 1
			i=$((i + 1))
		done
		# The token works once; don't leave it on disk.
		set_env VPS_PILOT_TOKEN
		[ -s "$STATE" ] ||
			die "the agent didn't enroll within 30 s. See: journalctl -u vps-pilot-agent"
	fi

	say "VPS Pilot agent $version is running."
	say "Logs: journalctl -u vps-pilot-agent -f"
}

# Everything is in main, so a download cut short can't run half a script.
main "$@"
