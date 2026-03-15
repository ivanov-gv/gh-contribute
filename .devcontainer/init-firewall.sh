#!/bin/bash
# init-firewall.sh — restrict outbound traffic to a known-good allowlist.
# This is the primary safety mechanism for --dangerously-skip-permissions mode:
# even if Claude Code runs unexpected commands, they cannot exfiltrate data or
# communicate with arbitrary hosts on the internet.
#
# Allowed outbound:
#   - GitHub (API, web, git, container registry)
#   - Anthropic API (Claude)
#   - npm registry
#   - Go module proxy and checksum DB
#   - Docker Hub and GHCR (pulling test images)
#   - Sentry, Statsig, VS Code Marketplace (Claude Code internals)
#   - DNS (UDP 53), SSH (TCP 22), localhost
#   - Host Docker network (so containers can communicate with the devcontainer)
set -euo pipefail
IFS=$'\n\t'

# 1. Preserve Docker's internal DNS NAT rules before flushing
DOCKER_DNS_RULES=$(iptables-save -t nat 2>/dev/null | grep "127\.0\.0\.11" || true)

# Flush all existing rules and ipsets
iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X
iptables -t mangle -F
iptables -t mangle -X
ipset destroy allowed-domains 2>/dev/null || true

# 2. Restore Docker DNS rules (needed for container-internal name resolution)
if [ -n "$DOCKER_DNS_RULES" ]; then
    echo "Restoring Docker DNS rules..."
    iptables -t nat -N DOCKER_OUTPUT 2>/dev/null || true
    iptables -t nat -N DOCKER_POSTROUTING 2>/dev/null || true
    echo "$DOCKER_DNS_RULES" | xargs -L 1 iptables -t nat
else
    echo "No Docker DNS rules to restore"
fi

# 3. Baseline rules: DNS, SSH, localhost
iptables -A OUTPUT -p udp --dport 53 -j ACCEPT
iptables -A INPUT  -p udp --sport 53 -j ACCEPT
iptables -A OUTPUT -p tcp --dport 22 -j ACCEPT
iptables -A INPUT  -p tcp --sport 22 -m state --state ESTABLISHED -j ACCEPT
iptables -A INPUT  -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT

# 4. Build the allowlist
ipset create allowed-domains hash:net

# GitHub IP ranges (web, API, git, packages)
echo "Fetching GitHub IP ranges..."
GH_META=$(curl -s https://api.github.com/meta)
if [ -z "$GH_META" ]; then
    echo "ERROR: Failed to fetch GitHub IP ranges"
    exit 1
fi
if ! echo "$GH_META" | jq -e '.web and .api and .git' >/dev/null; then
    echo "ERROR: GitHub API response missing required fields"
    exit 1
fi
echo "Processing GitHub IPs..."
while read -r cidr; do
    if [[ ! "$cidr" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}/[0-9]{1,2}$ ]]; then
        echo "ERROR: Invalid CIDR from GitHub meta: $cidr"
        exit 1
    fi
    echo "  Adding GitHub range $cidr"
    ipset add allowed-domains "$cidr"
done < <(echo "$GH_META" | jq -r '(.web + .api + .git + .packages)[]' | aggregate -q)

# Resolve individual allowed domains
for domain in \
    "api.anthropic.com" \
    "registry.npmjs.org" \
    "proxy.golang.org" \
    "sum.golang.org" \
    "storage.googleapis.com" \
    "registry-1.docker.io" \
    "auth.docker.io" \
    "index.docker.io" \
    "production.cloudflare.docker.com" \
    "ghcr.io" \
    "pkg-containers.githubusercontent.com" \
    "objects.githubusercontent.com" \
    "sentry.io" \
    "statsig.anthropic.com" \
    "statsig.com" \
    "marketplace.visualstudio.com" \
    "vscode.blob.core.windows.net" \
    "update.code.visualstudio.com" \
    "golang.org"; do
    echo "Resolving $domain..."
    ips=$(dig +noall +answer A "$domain" | awk '$4 == "A" {print $5}')
    if [ -z "$ips" ]; then
        echo "WARNING: Failed to resolve $domain — skipping"
        continue
    fi
    while read -r ip; do
        if [[ ! "$ip" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
            echo "ERROR: Invalid IP from DNS for $domain: $ip"
            exit 1
        fi
        echo "  Adding $ip for $domain"
        ipset add allowed-domains "$ip" 2>/dev/null || true  # duplicates are fine
    done < <(echo "$ips")
done

# 5. Allow the host Docker network so the devcontainer can reach test containers
HOST_IP=$(ip route | grep default | cut -d" " -f3)
if [ -z "$HOST_IP" ]; then
    echo "ERROR: Failed to detect host IP"
    exit 1
fi
HOST_NETWORK=$(echo "$HOST_IP" | sed "s/\.[0-9]*$/.0\/24/")
echo "Host network: $HOST_NETWORK"
iptables -A INPUT  -s "$HOST_NETWORK" -j ACCEPT
iptables -A OUTPUT -d "$HOST_NETWORK" -j ACCEPT

# 6. Set DROP as default policy, then allow established + allowlisted traffic
iptables -P INPUT   DROP
iptables -P FORWARD DROP
iptables -P OUTPUT  DROP

iptables -A INPUT  -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A OUTPUT -m set --match-set allowed-domains dst -j ACCEPT

# Hard reject everything else with immediate feedback (no silent drops)
iptables -A OUTPUT -j REJECT --reject-with icmp-admin-prohibited

# 7. Verify: blocked and allowed hosts
echo "Verifying firewall..."
if curl --connect-timeout 5 https://example.com >/dev/null 2>&1; then
    echo "ERROR: Firewall leak — reached https://example.com"
    exit 1
fi
echo "  PASS: example.com blocked"

if ! curl --connect-timeout 5 https://api.github.com/zen >/dev/null 2>&1; then
    echo "ERROR: Firewall too strict — cannot reach https://api.github.com"
    exit 1
fi
echo "  PASS: api.github.com reachable"

echo "Firewall configuration complete."
