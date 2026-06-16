#!/bin/bash

# Firecracker Setup and Run Script for Oblak
# This script prepares Firecracker resources and starts the API server

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STORAGE_DIR="$PROJECT_ROOT/storage"
FIRECRACKER_WS="$HOME/firecracker-workspace"

echo "=========================================="
echo "Oblak Firecracker Setup & Launch"
echo "=========================================="
echo

# Step 1: Verify Firecracker installation
echo "[1/5] Checking Firecracker installation..."
if ! command -v firecracker &> /dev/null; then
    echo "ERROR: Firecracker not found in PATH"
    echo "Install with: sudo apt-get install firecracker"
    exit 1
fi
FC_VERSION=$(firecracker --version | head -1)
echo "✓ Firecracker installed: $FC_VERSION"
echo

# Step 2: Verify kernel and rootfs
echo "[2/5] Checking kernel and rootfs..."
if [ ! -f "$FIRECRACKER_WS/vmlinux" ]; then
    echo "ERROR: Kernel not found at $FIRECRACKER_WS/vmlinux"
    echo "Create with: dd if=/dev/zero of=$FIRECRACKER_WS/vmlinux bs=1M count=20"
    exit 1
fi
echo "✓ Kernel: $FIRECRACKER_WS/vmlinux ($(du -h "$FIRECRACKER_WS/vmlinux" | cut -f1))"

if [ ! -f "$FIRECRACKER_WS/rootfs.ext4" ]; then
    echo "ERROR: Rootfs not found at $FIRECRACKER_WS/rootfs.ext4"
    exit 1
fi
echo "✓ Rootfs: $FIRECRACKER_WS/rootfs.ext4 ($(du -h "$FIRECRACKER_WS/rootfs.ext4" | cut -f1))"
echo

# Step 3: Verify /dev/kvm
echo "[3/5] Checking KVM availability..."
if [ ! -e /dev/kvm ]; then
    echo "ERROR: /dev/kvm not found"
    echo "Nested virtualization not available"
    echo "Check WSL .wslconfig: nestedVirtualization=true"
    exit 1
fi
if [ ! -r /dev/kvm ] || [ ! -w /dev/kvm ]; then
    echo "⚠ WARNING: /dev/kvm exists but not readable/writable"
    echo "Run: sudo usermod -a -G kvm $USER && newgrp kvm"
fi
echo "✓ /dev/kvm is available"
echo

# Step 4: Copy resources to project storage
echo "[4/5] Copying Firecracker resources..."
mkdir -p "$STORAGE_DIR"
cp "$FIRECRACKER_WS/vmlinux" "$STORAGE_DIR/vmlinux"
cp "$FIRECRACKER_WS/rootfs.ext4" "$STORAGE_DIR/rootfs.ext4"
echo "✓ Resources copied to $STORAGE_DIR"
echo

# Step 5a: Check for Go installation
echo "[5/5] Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo "⚠ Go not found in PATH"
    echo "Installing Go locally..."
    
    GO_VERSION="1.23.0"
    GOARCH=$(uname -m)
    if [ "$GOARCH" = "x86_64" ]; then
        GOARCH="amd64"
    elif [ "$GOARCH" = "aarch64" ]; then
        GOARCH="arm64"
    fi
    GOOS=$(uname -s | tr '[:upper:]' '[:lower:]')
    
    GO_ARCHIVE="go${GO_VERSION}.${GOOS}-${GOARCH}.tar.gz"
    GO_URL="https://go.dev/dl/${GO_ARCHIVE}"
    GO_LOCAL_PATH="$STORAGE_DIR/go"
    
    mkdir -p "$GO_LOCAL_PATH"
    echo "  Downloading $GO_ARCHIVE..."
    cd "$STORAGE_DIR"
    
    if command -v wget &> /dev/null; then
        wget -q "$GO_URL" -O "$GO_ARCHIVE"
    elif command -v curl &> /dev/null; then
        curl -s -L "$GO_URL" -o "$GO_ARCHIVE"
    else
        echo "ERROR: Neither wget nor curl found"
        exit 1
    fi
    
    echo "  Extracting..."
    tar -xzf "$GO_ARCHIVE" -C "$GO_LOCAL_PATH" --strip-components=1
    rm "$GO_ARCHIVE"
    
    export PATH="$GO_LOCAL_PATH/bin:$PATH"
    echo "✓ Go installed locally at $GO_LOCAL_PATH"
else
    echo "✓ Go is installed: $(go version | awk '{print $3}')"
fi
echo

# Step 5b: Start server
echo "Starting API server with Firecracker runner..."
echo

# Export Firecracker paths
export FIRECRACKER_KERNEL="$STORAGE_DIR/vmlinux"
export FIRECRACKER_ROOTFS="$STORAGE_DIR/rootfs.ext4"
export OBLAK_RUNS_DIR="${OBLAK_RUNS_DIR:-/tmp/oblak-runs}"
export OBLAK_ADDR="${OBLAK_ADDR:-127.0.0.1:8080}"
export OBLAK_DB="$STORAGE_DIR/oblak.db"

echo "Environment:"
echo "  FIRECRACKER_KERNEL=$FIRECRACKER_KERNEL"
echo "  FIRECRACKER_ROOTFS=$FIRECRACKER_ROOTFS"
echo "  OBLAK_RUNS_DIR=$OBLAK_RUNS_DIR"
echo "  OBLAK_ADDR=$OBLAK_ADDR"
echo "  OBLAK_DB=$OBLAK_DB"
echo "  PATH includes: $(dirname "$(command -v go)")"
echo

# Create runs directory
mkdir -p "$STORAGE_DIR/runs"

# Start server
cd "$PROJECT_ROOT"
exec go run ./cmd/api/main.go
