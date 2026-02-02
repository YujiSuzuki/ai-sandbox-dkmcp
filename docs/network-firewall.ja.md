# ネットワーク制限（ファイアウォール）

AI Sandboxにネットワーク制限を追加する方法を説明します。

[← READMEに戻る](../README.ja.md)

---

## AI Sandboxのネットワーク制限について

AI Sandboxは秘匿ファイルの隠蔽とコンテナ間アクセスを提供しますが、**ネットワーク制限は含まれていません**。AI Sandboxのコンテナ内から外部への通信は制限されないため、必要に応じてファイアウォールの導入を検討してください。

## Anthropic公式のファイアウォールスクリプト

Anthropicは、Claude Code用のDevContainerにファイアウォールスクリプトを公開しています。

- **リポジトリ:** [anthropics/claude-code/.devcontainer](https://github.com/anthropics/claude-code/tree/main/.devcontainer)
- **スクリプト:** [init-firewall.sh](https://github.com/anthropics/claude-code/blob/main/.devcontainer/init-firewall.sh)

### 仕組みの概要

- `iptables` + `ipset` によるホワイトリスト方式
- デフォルトですべての外部通信をブロックし、許可されたドメインのみ通信を許可
- 許可対象: GitHub、npm registry、Anthropic API、VS Code Marketplace など

> **注意:** このスクリプトはAnthropic公式のものです。スクリプトの詳細や最新の変更については、[公式リポジトリ](https://github.com/anthropics/claude-code/tree/main/.devcontainer)を参照してください。

---

## AI Sandboxへの導入方法

以下は導入の一例です。公式スクリプトの仕様変更により手順が変わる可能性があります。

### Step 1: スクリプトをダウンロード

```bash
# プロジェクトルートで実行
curl -o .devcontainer/init-firewall.sh \
  https://raw.githubusercontent.com/anthropics/claude-code/main/.devcontainer/init-firewall.sh
chmod +x .devcontainer/init-firewall.sh
```

### Step 2: devcontainer.json にファイアウォール初期化を追加

`.devcontainer/devcontainer.json` の `postStartCommand` にスクリプトを追加します:

```jsonc
// 既存の postStartCommand の先頭に追加
"postStartCommand": "/workspace/.devcontainer/init-firewall.sh && /workspace/.sandbox/scripts/merge-claude-settings.sh && ..."
```

> **ヒント:** ファイアウォールは他のスクリプトより先に実行するのがよいでしょう。

### Step 3: Dockerfile に必要なパッケージを追加

ファイアウォールスクリプトは `iptables` と `ipset` を使用します。`.sandbox/Dockerfile` に追加してください:

```dockerfile
# ファイアウォール用パッケージ
RUN sudo apt-get update && sudo apt-get install -y iptables ipset curl \
    && sudo rm -rf /var/lib/apt/lists/*
```

> **注意:** スクリプトに `sudo` が必要な場合があります。AI Sandboxの `node` ユーザーには `iptables` の sudo 権限が付与されていないため、Dockerfileでの設定か、sudoersの調整が必要です。

### Step 4: DevContainerを再ビルド

```bash
# VS Code: Cmd+Shift+P → "Dev Containers: Rebuild Container"
```

---

## 注意事項

### DockMCPとの共存

DockMCPはホストOSへの通信（`host.docker.internal`）を使用します。公式スクリプトはホストネットワークへの通信を許可しているため、通常は問題なく共存できます。

もし接続に問題が出た場合は、ファイアウォールのルールで DockMCP のポート（デフォルト: 8080）へのアクセスが許可されているか確認してください。

### Claude Code 以外のAIツールを使う場合

公式スクリプトの許可リストは Claude Code 向けです。Gemini CLI や他のAIツールを使う場合は、そのツールが必要とするドメインを追加する必要があります。

### CLI Sandboxでの利用

CLI Sandbox（`cli_sandbox/`）で使う場合は、同様の設定を `cli_sandbox/docker-compose.yml` 側にも適用してください。

---

## 参考リンク

- [Anthropic公式 DevContainer](https://github.com/anthropics/claude-code/tree/main/.devcontainer) — ファイアウォールスクリプトの提供元
- [Claude Code サンドボックス化ドキュメント](https://code.claude.com/docs/ja/sandboxing) — Claude Code のネイティブサンドボックス機能
- [Docker Sandbox](https://docs.docker.com/ai/sandboxes) — Docker公式のAIサンドボックス（microVM方式）
