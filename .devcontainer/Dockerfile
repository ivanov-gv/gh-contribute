FROM golang:bookworm

ARG TZ
ENV TZ="$TZ"

ARG CLAUDE_CODE_VERSION=latest
ARG GIT_DELTA_VERSION=0.18.2
ARG ZSH_IN_DOCKER_VERSION=1.2.0
ARG NODE_VERSION=20

# Install all system-level tools via the dedicated script.
# To add a new tool, edit install-deps.sh — no Dockerfile changes needed.
COPY .devcontainer/install-deps.sh /tmp/install-deps.sh
RUN chmod +x /tmp/install-deps.sh && \
  GIT_DELTA_VERSION=${GIT_DELTA_VERSION} NODE_VERSION=${NODE_VERSION} \
  /tmp/install-deps.sh && \
  rm /tmp/install-deps.sh

# Create non-root dev user.
# adduser (Debian wrapper) picks a free UID automatically — avoids the
# "no matching entries in passwd file" error that occurs when a hard-coded
# UID like 1000 is already taken by a package installed above.
RUN adduser --disabled-password --gecos "" --shell /usr/bin/zsh dev

# Allow dev user to use docker group (GID reconciled at runtime in setup.sh)
RUN groupadd -f docker && usermod -aG docker dev

# npm global dir owned by dev so Claude Code installs without sudo
RUN mkdir -p /usr/local/share/npm-global && chown dev:dev /usr/local/share/npm-global

# Persist bash history across rebuilds
RUN mkdir /commandhistory && touch /commandhistory/.bash_history && chown -R dev /commandhistory

# Signal that this is a devcontainer
ENV DEVCONTAINER=true

# GOPATH in the dev user home; Go binary already on PATH via the base image
ENV GOPATH=/home/dev/go
ENV PATH=$PATH:/home/dev/go/bin

# Create workspace and per-user config directories
RUN mkdir -p /workspace /home/dev/.claude /home/dev/go && \
  chown -R dev:dev /workspace /home/dev/.claude /home/dev/go

WORKDIR /workspace

# Build the gh-contribute binary from project source and install system-wide
COPY . /build/gh-contribute
RUN cd /build/gh-contribute && \
  go build -o /usr/local/bin/gh-contribute ./cmd/gh-contribute && \
  rm -rf /build/gh-contribute

RUN chown -R dev:dev /workspace /home/dev/.claude /home/dev/go

USER dev

# npm global config — installs packages to a dir the dev user owns
ENV NPM_CONFIG_PREFIX=/usr/local/share/npm-global
ENV PATH=$PATH:/usr/local/share/npm-global/bin
ENV SHELL=/bin/zsh
ENV EDITOR=nano
ENV VISUAL=nano

# Install zsh with Powerline10k theme
RUN sh -c "$(wget -O- https://github.com/deluan/zsh-in-docker/releases/download/v${ZSH_IN_DOCKER_VERSION}/zsh-in-docker.sh)" -- \
  -p git \
  -p fzf \
  -a "source /usr/share/doc/fzf/examples/key-bindings.zsh" \
  -a "source /usr/share/doc/fzf/examples/completion.zsh" \
  -a "export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
  -x

# Install Claude Code globally
RUN npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}

# Copy setup scripts and grant passwordless sudo for them
COPY .devcontainer/init-firewall.sh /usr/local/bin/
COPY .devcontainer/setup.sh /usr/local/bin/
USER root
RUN chmod +x /usr/local/bin/init-firewall.sh /usr/local/bin/setup.sh && \
  echo "dev ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh, /usr/local/bin/setup.sh" \
    > /etc/sudoers.d/dev-scripts && \
  chmod 0440 /etc/sudoers.d/dev-scripts
USER dev
