#!/usr/bin/env bash
# Jednokratno: preuzmi ClamAV virus definicije u storage/clamav/database.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DB_DIR="$ROOT/storage/clamav/database"
CONF="$ROOT/storage/clamav/freshclam.conf"

if ! command -v freshclam >/dev/null 2>&1; then
	echo "freshclam not found. Install ClamAV first, e.g.:" >&2
	echo "  sudo apt install clamav clamav-freshclam   # Debian/Ubuntu" >&2
	echo "  sudo dnf install clamav clamav-freshclam   # Fedora" >&2
	exit 1
fi

mkdir -p "$DB_DIR"

cat >"$CONF" <<EOF
DatabaseDirectory $DB_DIR
DatabaseMirror database.clamav.net
EOF

echo "Downloading virus definitions to:"
echo "  $DB_DIR"
freshclam --config-file="$CONF"

echo "OK. clamscan version:"
clamscan -d "$DB_DIR" --version
