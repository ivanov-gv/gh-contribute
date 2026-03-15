#!/bin/bash
# setup.sh — run as root on container start (postStartCommand).
#
# Steps:
#   1. Create an isolated Docker bridge network for this devcontainer instance.
#      All test containers started by Claude Code must join this network so they
#      are reachable from the devcontainer but invisible to other devcontainers
#      running on the same host Docker daemon.
#   2. Grant the dev user access to the Docker socket if it exists.
#   3. Apply the outbound firewall (delegates to init-firewall.sh).
set -euo pipefail
IFS=$'\n\t'

# ── 1. Isolated Docker network ─────────────────────────────────────────────────

# DOCKER_NETWORK is set in devcontainer.json containerEnv.
# Fall back to hostname-based name if the env var is somehow missing.
NETWORK_NAME="${DOCKER_NETWORK:-gh-contribute-$(hostname)}"

if docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
    echo "Docker network '$NETWORK_NAME' already exists — skipping creation"
else
    docker network create \
        --driver bridge \
        --label "devcontainer.id=$(hostname)" \
        "$NETWORK_NAME"
    echo "Created Docker network: $NETWORK_NAME"
fi

# Write the network name to a file readable by the dev user.
# This lets shell scripts source it, in addition to the env var.
echo "export DOCKER_NETWORK=$NETWORK_NAME" > /etc/profile.d/docker-network.sh
chmod 644 /etc/profile.d/docker-network.sh

# ── 2. Docker socket permissions ──────────────────────────────────────────────

DOCKER_SOCK=/var/run/docker.sock
if [ -S "$DOCKER_SOCK" ]; then
    # Get the GID of the socket and add dev user to that group
    DOCKER_GID=$(stat -c '%g' "$DOCKER_SOCK")
    if ! getent group "$DOCKER_GID" >/dev/null 2>&1; then
        groupadd -g "$DOCKER_GID" docker-host
    fi
    DOCKER_GROUP=$(getent group "$DOCKER_GID" | cut -d: -f1)
    usermod -aG "$DOCKER_GROUP" dev
    echo "Added dev user to group '$DOCKER_GROUP' (gid=$DOCKER_GID) for Docker socket access"
else
    echo "WARNING: Docker socket not found at $DOCKER_SOCK — Docker-in-Docker will not work"
fi

# ── 3. Firewall ────────────────────────────────────────────────────────────────

echo "Initialising firewall..."
/usr/local/bin/init-firewall.sh

echo ""
echo "Setup complete."
echo "  Docker network : $NETWORK_NAME"
echo "  Use flag       : docker run --network \"\$DOCKER_NETWORK\" <image>"
echo "  Firewall       : outbound restricted to allowlist"
