FROM golang:1.26-bookworm

# System-Tools die Claude Code braucht
RUN apt-get update && apt-get install -y \
    git \
    curl \
    sudo \
    ripgrep \
    fd-find \
    jq \
    tree \
    htop \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Node.js 20 LTS (Claude Code braucht Node)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Claude Code installieren
RUN npm install -g @anthropic-ai/claude-code

# Nicht-Root-User erstellen
RUN useradd -m -s /bin/bash claude \
    && echo "claude ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Git-Config damit Claude committen kann
RUN git config --system user.name "Claude Code" \
    && git config --system user.email "claude@autonomous"

USER claude
WORKDIR /workspace

# Go-Module-Cache im Container
ENV GOPATH=/home/claude/go
ENV PATH="${GOPATH}/bin:${PATH}"
