#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOTFS="${1:-$HOME/firecracker-workspace/rootfs.ext4}"
MOUNT_DIR="${MOUNT_DIR:-/mnt/oblak-rootfs}"
AGENT_SRC="$PROJECT_ROOT/internal/runner/firecracker/guest-agent.py"

if [[ ! -f "$ROOTFS" ]]; then
  echo "rootfs not found: $ROOTFS" >&2
  exit 1
fi

if [[ ! -f "$AGENT_SRC" ]]; then
  echo "guest agent not found: $AGENT_SRC" >&2
  exit 1
fi

cleanup() {
  if mountpoint -q "$MOUNT_DIR"; then
    sudo umount "$MOUNT_DIR"
  fi
}
trap cleanup EXIT

sudo mkdir -p "$MOUNT_DIR"
sudo mount -o loop "$ROOTFS" "$MOUNT_DIR"

sudo mkdir -p "$MOUNT_DIR/usr/local/bin" "$MOUNT_DIR/etc/systemd/system" "$MOUNT_DIR/function"
sudo cp "$AGENT_SRC" "$MOUNT_DIR/usr/local/bin/oblak-guest-agent.py"
sudo chmod 0755 "$MOUNT_DIR/usr/local/bin/oblak-guest-agent.py"

if [[ ! -e "$MOUNT_DIR/usr/bin/python3" ]]; then
  echo "warning: python3 not found in rootfs; install python3 in the rootfs before invoking functions" >&2
fi

if [[ -d "$MOUNT_DIR/etc/systemd/system" ]]; then
  sudo tee "$MOUNT_DIR/etc/systemd/system/oblak-guest-agent.service" >/dev/null <<'SERVICE'
[Unit]
Description=Oblak Firecracker guest agent
After=multi-user.target

[Service]
Type=simple
ExecStart=/usr/bin/python3 /usr/local/bin/oblak-guest-agent.py
Restart=always
RestartSec=1
StandardOutput=journal+console
StandardError=journal+console

[Install]
WantedBy=multi-user.target
SERVICE

  sudo mkdir -p "$MOUNT_DIR/etc/systemd/system/multi-user.target.wants"
  sudo ln -sf ../oblak-guest-agent.service \
    "$MOUNT_DIR/etc/systemd/system/multi-user.target.wants/oblak-guest-agent.service"
fi

sync
echo "installed guest agent into $ROOTFS"
