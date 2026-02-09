FROM node:20-slim

# Install required tools
# 必要なツールをインストール
RUN apt-get update && apt-get install -y \
    ca-certificates \
    git \
    python3 \
    python3-pip \
    vim \
    less \
    sudo \
    curl \
    wget \
    iputils-ping \
    dnsutils \
    jq \
    # Process utilities (ps, top, kill commands) / プロセス管理ツール (ps, top, kill コマンド)
    procps \
    # Advanced network debugging tools (uncomment if needed):
    # ネットワーク詳細デバッグツール（必要に応じてコメント解除）:
    # netcat-openbsd \   # Port connectivity testing / ポート到達性テスト
    # iproute2 \         # Modern network tools (ip, ss commands) / モダンなネットワークツール (ip, ss コマンド)
    && rm -rf /var/lib/apt/lists/*

# Install Go (for SandboxMCP and other Go tools)
# Go をインストール（SandboxMCP と他の Go ツール用）
ARG GO_VERSION=1.24.12
ARG TARGETARCH
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz" | tar -C /usr/local -xz
ENV GOPATH="/home/node/go"
ENV PATH="/usr/local/go/bin:${GOPATH}/bin:${PATH}"

# Install Claude Code
# Claude Code インストール
RUN npm install -g @anthropic-ai/claude-code

# Install Gemini CLI
# Gemini CLI インストール
RUN npm install -g @google/gemini-cli

# Configure sudoers (allow only package managers)
# ⚠️ SECURITY: npm/pip3 sudo can be exploited via malicious packages.
#    See README.md "Security Features" for mitigation options.
# sudoers設定 ⚠️ npm/pip3のsudoは悪意あるパッケージで悪用される可能性あり（詳細はREADME参照）
RUN echo "node ALL=(ALL) NOPASSWD: /usr/bin/apt-get" > /etc/sudoers.d/node && \
    echo "node ALL=(ALL) NOPASSWD: /usr/bin/apt" >> /etc/sudoers.d/node && \
    echo "node ALL=(ALL) NOPASSWD: /usr/bin/dpkg" >> /etc/sudoers.d/node && \
    echo "node ALL=(ALL) NOPASSWD: /usr/bin/pip3" >> /etc/sudoers.d/node && \
    echo "node ALL=(ALL) NOPASSWD: /usr/local/bin/npm" >> /etc/sudoers.d/node && \
    chmod 0440 /etc/sudoers.d/node

# Prepare node user home directory
# node ユーザーのホームディレクトリ準備
RUN mkdir -p /home/node/.claude /home/node/.local/bin \
    && chown -R node:node /home/node

# Switch to node user
# node ユーザーに切り替え
USER node

# Add .local/bin to PATH for persistent tool installations
# 永続化されたツールインストール用に.local/binをPATHに追加
ENV PATH="/home/node/.local/bin:${PATH}"

WORKDIR /workspace
CMD ["bash"]
