FROM golang:latest

ARG TZ
ENV TZ="$TZ"

ARG GIT_DELTA_VERSION=0.18.2
ARG ZSH_IN_DOCKER_VERSION=1.2.0

# Install all system-level tools via the dedicated script.
# To add a new tool, edit install-deps.sh — no Dockerfile changes needed.
COPY .devcontainer/claude-code/install-deps.sh /tmp/install-deps.sh
RUN chmod +x /tmp/install-deps.sh && \
  GIT_DELTA_VERSION=${GIT_DELTA_VERSION} \
  /tmp/install-deps.sh && \
  rm /tmp/install-deps.sh

# Create non-root dev user.
RUN adduser --disabled-password --gecos "" --shell /usr/bin/zsh dev &&  \
    chown dev:dev /home/dev &&  \
    groupadd -f docker && usermod -aG docker dev

# Persist bash history across rebuilds
RUN mkdir /commandhistory && touch /commandhistory/.bash_history && chown -R dev /commandhistory

# Copy setup scripts and grant passwordless sudo for them
COPY --chmod=755 .devcontainer/shared/init-firewall.sh .devcontainer/shared/setup.sh /usr/local/bin/
RUN echo "dev ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh, /usr/local/bin/setup.sh" \
    > /etc/sudoers.d/dev-scripts && \
  chmod 0440 /etc/sudoers.d/dev-scripts

USER dev

ENV GOPATH=/home/dev/go
ENV PATH=$PATH:/home/dev/go/bin
ENV PATH=$PATH:/home/dev/.local/bin
ENV SHELL=/bin/zsh
ENV EDITOR=nano
ENV VISUAL=nano

COPY --chmod=755 .devcontainer/claude-code/install-user-deps.sh /usr/local/bin/
RUN /usr/local/bin/install-user-deps.sh

WORKDIR /workspace
RUN chown -R dev:dev /workspace

COPY --chown=dev:dev . ./tmp/gh-contribute
RUN cd ./tmp/gh-contribute &&  \
    go install ./cmd/gh-contribute && \
    cd ../.. && rm -rf ./tmp
