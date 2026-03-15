# Devcontainer Configuration Design

**Date:** 2026-03-15
**Status:** Approved

## Overview

Replace the existing single VS Code devcontainer with three purpose-built configurations:

1. **`claude-code`** — JetBrains GoLand + Claude Code CLI, suitable for `--dangerously-skip-permissions` mode
2. **`default`** — JetBrains GoLand, minimal Go development environment
3. **`claude-session`** — Standalone Docker image for isolated Claude Code terminal sessions

---

## File Structure

```
.devcontainer/
  claude-code/
    devcontainer.json     ← JetBrains + Claude Code
    Dockerfile
    install-deps.sh       ← system deps, no Node.js
  default/
    devcontainer.json     ← JetBrains minimal (rewritten)
  shared/
    setup.sh              ← moved from root
    init-firewall.sh      ← moved from root

deploy/
  claude-session/
    Dockerfile            ← same image as claude-code
    docker-compose.yml    ← usage entry point
```

The root `.devcontainer/devcontainer.json` and `.devcontainer/Dockerfile` are removed and replaced by the named configs above.

---

## Variant 1: `claude-code`

### Purpose

Full development environment for JetBrains GoLand with Claude Code CLI. Designed to be safe for `--dangerously-skip-permissions` mode via an outbound firewall allowlist.

### Dockerfile

- **Base image:** `golang:latest` (Debian Bookworm, always latest Go)
- **No Node.js, no npm** — Claude Code installed as a native standalone binary, downloaded from official GitHub releases with architecture detection at build time
- **gopls** installed via `go install golang.org/x/tools/gopls@latest`
- **Dev user:** non-root `dev` user with zsh as default shell
- **Shell:** zsh + Powerline10k theme (via zsh-in-docker), fzf key bindings
- **Tools:** git, git-delta, Docker CLI (no daemon), docker-compose-plugin, iptables, ipset, iproute2, dnsutils, aggregate, jq, gh, nano, vim, sudo
- **Firewall scripts** copied to `/usr/local/bin/` with passwordless sudo for `dev`
- **History persistence:** `/commandhistory` directory owned by `dev`

### install-deps.sh

Same as current but with the Node.js section removed entirely.

### devcontainer.json

| Field | Value |
|---|---|
| `name` | `gh-contribute (claude-code)` |
| `build.dockerfile` | `Dockerfile` |
| `build.context` | `../..` |
| `build.args` | `TZ`, `CLAUDE_CODE_VERSION`, `GIT_DELTA_VERSION`, `ZSH_IN_DOCKER_VERSION` |
| `runArgs` | `--cap-add=NET_ADMIN`, `--cap-add=NET_RAW`, Docker socket bind mount |
| `remoteUser` | `dev` |
| `workspaceMount` | bind, delegated consistency |
| `workspaceFolder` | `/workspace` |
| `postStartCommand` | `sudo /usr/local/bin/setup.sh` |
| `waitFor` | `postStartCommand` |

**JetBrains customization:**
```json
"customizations": {
  "jetbrains": {
    "backend": "GoLand"
  }
}
```

**Mounts:**
- History volume: `gh-contribute-bashhistory-${devcontainerId}` → `/commandhistory`
- `~/.claude` bind mount: `${localEnv:HOME}/.claude` → `/home/dev/.claude` — imports local Claude Code plugins, settings, and auth into the container

**containerEnv:**
- `CLAUDE_CONFIG_DIR=/home/dev/.claude`
- `ANTHROPIC_API_KEY=${localEnv:ANTHROPIC_API_KEY}`
- `GH_CONTRIBUTE_TOKEN=${localEnv:GH_CONTRIBUTE_TOKEN}`
- `GITHUB_TOKEN=${localEnv:GITHUB_TOKEN}`
- `GH_TOKEN=${localEnv:GH_TOKEN}`
- `DOCKER_NETWORK=gh-contribute-${devcontainerId}`
- `POWERLEVEL9K_DISABLE_GITSTATUS=true`

---

## Variant 2: `default`

### Purpose

Minimal Go development environment for JetBrains GoLand. No Claude Code, no firewall, no custom Dockerfile.

### devcontainer.json

| Field | Value |
|---|---|
| `name` | `gh-contribute (default)` |
| `image` | `golang:latest` |
| `remoteUser` | `root` |
| `postCreateCommand` | `go install golang.org/x/tools/gopls@latest` |

**JetBrains customization:**
```json
"customizations": {
  "jetbrains": {
    "backend": "GoLand"
  }
}
```

**containerEnv:**
- `GOPATH=/root/go`
- `PATH` extended with `/root/go/bin`

No mounts, no runArgs, no firewall, no extra tools. Git is already present in the base image.

---

## Variant 3: `claude-session`

### Purpose

Standalone Docker image for isolated Claude Code terminal sessions. Not a devcontainer — started manually via Docker Compose, used interactively via `docker compose run`.

### Dockerfile

Identical content to `claude-code/Dockerfile`. Same base, same tools, same Claude Code native binary, same `dev` user and zsh setup. Firewall scripts present and sudoable, but not auto-run.

### docker-compose.yml

```yaml
services:
  claude-session:
    build: .
    stdin_open: true
    tty: true
    cap_add:
      - NET_ADMIN
      - NET_RAW
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
```

**Usage:**
```bash
docker compose run --rm claude-session
```

Drops into zsh as `dev`. Clone a repo, run `sudo init-firewall.sh` for outbound protection, then run `claude`.

No workspace bind mount (fresh clone inside container), no `~/.claude` bind mount (paths differ), no API key passthrough (authenticate inside container via `claude auth login`).

---

## Shared Scripts

`setup.sh` and `init-firewall.sh` move from `.devcontainer/` root to `.devcontainer/shared/`. The `claude-code` Dockerfile copies them from there. The `claude-session` Dockerfile does the same. The `default` variant does not use them.

`init-firewall.sh` allowlist stays unchanged:
- GitHub IP ranges
- `api.anthropic.com`
- Go module proxy / checksum DB
- npm registry (kept — useful even without Node.js, Claude Code may use it)
- Docker Hub, GHCR
- Sentry, Statsig, VS Code Marketplace (Claude Code internals)
- DNS, SSH, localhost, host Docker network

---

## Migration

| Old path | New path |
|---|---|
| `.devcontainer/devcontainer.json` | `.devcontainer/claude-code/devcontainer.json` |
| `.devcontainer/Dockerfile` | `.devcontainer/claude-code/Dockerfile` |
| `.devcontainer/install-deps.sh` | `.devcontainer/claude-code/install-deps.sh` |
| `.devcontainer/setup.sh` | `.devcontainer/shared/setup.sh` |
| `.devcontainer/init-firewall.sh` | `.devcontainer/shared/init-firewall.sh` |
| `.devcontainer/default/devcontainer.json` | `.devcontainer/default/devcontainer.json` (rewritten) |
| _(new)_ | `deploy/claude-session/Dockerfile` |
| _(new)_ | `deploy/claude-session/docker-compose.yml` |
