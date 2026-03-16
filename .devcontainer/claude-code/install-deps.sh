#!/bin/bash
# install-deps.sh — install system-level tools during image build.
# Runs as root. Edit this file to add new tools without touching the Dockerfile.
set -euo pipefail

# ── System packages ────────────────────────────────────────────────────────────
apt-get update && apt-get install -y --no-install-recommends \
  less \
  git \
  procps \
  sudo \
  fzf \
  zsh \
  man-db \
  unzip \
  gnupg2 \
  curl \
  wget \
  iptables \
  ipset \
  iproute2 \
  dnsutils \
  aggregate \
  jq \
  nano \
  vim \
  gh \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

# ── Docker CLI ─────────────────────────────────────────────────────────────────
# No daemon installed — the devcontainer uses the host Docker socket via mount.
# docker-compose-plugin provides the `docker compose` (v2) subcommand as well.
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
# Detect Debian codename dynamically so this works across Debian releases.
DISTRO_CODENAME=$(. /etc/os-release && echo "$VERSION_CODENAME")
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
https://download.docker.com/linux/debian ${DISTRO_CODENAME} stable" \
  > /etc/apt/sources.list.d/docker.list
apt-get update
apt-get install -y --no-install-recommends docker-ce-cli docker-compose-plugin
apt-get clean && rm -rf /var/lib/apt/lists/*

# ── git-delta ──────────────────────────────────────────────────────────────────
# Improved diff viewer used by git.
ARCH=$(dpkg --print-architecture)
GIT_DELTA_VERSION="${GIT_DELTA_VERSION:-0.18.2}"
wget -q "https://github.com/dandavison/delta/releases/download/${GIT_DELTA_VERSION}/git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb"
dpkg -i "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb"
rm "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb"

# ── Add more tools below ───────────────────────────────────────────────────────
# Example: apt-get install -y --no-install-recommends <package>
# Example: curl -fsSL <url> | tar -C /usr/local/bin -xz <binary>
