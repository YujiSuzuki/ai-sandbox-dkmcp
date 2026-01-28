# DockMCP

[English README is here](README.md)

**AIコーディングアシスタントのための安全なDockerコンテナアクセス**

DockMCPは、ホストOS上で動作し、DevContainer内のAIアシスタント（Claude Code、Gemini Code Assistなど）が他のDockerコンテナのログ確認やテスト実行を安全に行えるようにするMCPサーバーです。

DockMCPが解決する課題やアーキテクチャの全体像については [ルートの README.ja.md](../README.ja.md) を参照してください。

---

## 目次

- [機能](#機能)
- [インストール](#インストール)
- [サーバー起動](#サーバー起動)
- [AIアシスタントとの接続](#aiアシスタントとの接続)
- [CLIコマンド](#cliコマンド)
- [セキュリティモード](#セキュリティモード)
- [認証](#認証)
- [設定リファレンス](#設定リファレンス)
  - [ファイルアクセスブロック（blocked_paths）](#ファイルアクセスブロックblocked_paths)
  - [出力マスキング](#出力マスキング)
  - [ホストパスマスキング](#ホストパスマスキング)
  - [パーミッション](#パーミッション)
  - [デフォルトコマンド](#デフォルトコマンドexec_whitelist-)
  - [危険モード（exec_dangerously）](#危険モードexec_dangerously)
- [アーキテクチャ](#アーキテクチャ)
- [設計思想](#設計思想)
- [提供されるMCPツール](#提供されるmcpツール)
- [トラブルシューティング](#トラブルシューティング)
- [ライセンス](#ライセンス)

---

## 機能

- 🔒 **セキュリティ第一の設計** - ホワイトリストベースのコンテナとコマンドアクセス
- 🤖 **マルチAIサポート** - Claude Code、Gemini Code Assistで動作
- 🚀 **依存関係ゼロ** - 単一バイナリ、ランタイム要件なし
- 🌐 **クロスプラットフォーム** - Windows、macOS（Intel & Apple Silicon）、Linux
- 📝 **監査ログ** - コンプライアンス対応のため全操作を記録可能
- ⚡ **MCP標準** - 将来の拡張性を見据えてMCPを採用

## インストール

ホストOS上で実行してください。

**macOS (Apple Silicon)**
```bash
curl -L https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases/latest/download/dkmcp_darwin_arm64 -o dkmcp
chmod +x dkmcp
sudo mv dkmcp /usr/local/bin/
```

**macOS (Intel)**
```bash
curl -L https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases/latest/download/dkmcp_darwin_amd64 -o dkmcp
chmod +x dkmcp
sudo mv dkmcp /usr/local/bin/
```

**Windows**
1. [Releases](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases)から `dkmcp_windows_amd64.exe` をダウンロード
2. フォルダに配置（例: `C:\tools\`）
3. PATHに追加するか、フルパスで使用

**Linux**
```bash
curl -L https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases/latest/download/dkmcp_linux_amd64 -o dkmcp
chmod +x dkmcp
sudo mv dkmcp /usr/local/bin/
```

**ソースからビルド**
```bash
cd dkmcp
make install  # ~/go/bin/ にインストール
```

## サーバー起動

### 設定ファイルの準備

```bash
# サンプル設定をコピー
cp configs/dkmcp.example.yaml dkmcp.yaml

# コンテナ名に合わせて編集
nano dkmcp.yaml
```

設定例:
```yaml
server:
  port: 8080
  host: "127.0.0.1"

security:
  mode: "moderate"  # strict, moderate, または permissive

  allowed_containers:
    - "myapp-*"
    - "mydb-*"

  exec_whitelist:
    "myapp-api":
      - "npm test"
      - "pytest /app/tests"
    "*":
      - "pwd"
```

### 起動

```bash
# ホストOSで実行
dkmcp serve --config dkmcp.yaml
```

以下のような出力が表示されます:
```
2026-01-22 12:55:17 INFO  Starting DockMCP server version=dev security_mode=moderate port=8080 log_level=info
2026-01-22 12:55:17 INFO  Found accessible containers count=3
2026-01-22 12:55:17 INFO  MCP server listening url=http://127.0.0.1:8080 health_check=http://127.0.0.1:8080/health sse_endpoint=http://127.0.0.1:8080/sse
2026-01-22 12:55:17 INFO  Press Ctrl+C to stop
```

### Verbosity レベル

デバッグ用にログの詳細度を上げるには `-v` フラグを使用します：

```bash
dkmcp serve --config dkmcp.yaml -v      # レベル1: JSONリクエスト/レスポンス出力
dkmcp serve --config dkmcp.yaml -vv     # レベル2: DEBUGレベル + JSON出力
dkmcp serve --config dkmcp.yaml -vvv    # レベル3: フルデバッグ（全ノイズ表示）
```

| レベル | フラグ | 説明 |
|--------|--------|------|
| 0 | (なし) | 通常のINFOレベル、最小出力 |
| 1 | `-v` | JSONリクエスト/レスポンス表示、未初期化接続をフィルタ |
| 2 | `-vv` | DEBUGレベル + JSON出力、未初期化接続をフィルタ |
| 3 | `-vvv` | フルデバッグ、ノイズを含む全接続を表示 |

**注:** 「ノイズ」とは未初期化のSSE接続（例：VS Code拡張機能のプローブ）を指します。レベル0-2ではこれらをフィルタしてログをきれいに保ちます。

### ログ機能

**リクエスト番号:** 各リクエストには一意の番号 `[#N]` が割り当てられ、複数のリクエストが同時に処理される際の追跡に使えます：

```
═══ [#1] ═══════════════════════════════════════════════════════════
▼ REQUEST client=claude-code method=tools/call tool=list_containers id=2
...
═══ [#1] ═══════════════════════════════════════════════════════════
```

**クライアント識別:** サーバーはログにクライアント名（MCP `clientInfo` から取得）を表示します：
- `claude-code` - Claude Code拡張機能
- `dkmcp-go-client` - DockMCP CLIクライアント（`--client-suffix` でカスタムサフィックス対応）

**グレースフルシャットダウン:** サーバー停止時（Ctrl+C）：
- アクティブな接続が閉じるまで最大2秒待機
- タイムアウト後は残りの接続を強制クローズ
- 未初期化接続のUser-Agentサマリーを表示：
  ```
  Uninitialized connection summary: claude-code/2.1.7: 81, node: 1
  ```

### 複数インスタンスの起動

ポートと設定ファイルを分けることで、用途別に複数のDockMCPサーバーを同時に起動できます：

```bash
# 開発インスタンス（寛容）
dkmcp serve --port 8080 --config dev.yaml

# 別プロジェクト用（厳格）
dkmcp serve --port 8081 --config strict.yaml
```

## AIアシスタントとの接続

DevContainer内でのMCP設定手順は [ルートの README.ja.md「クイックスタート」](../README.ja.md#オプションb-sandbox--dockmcp) を参照してください。

設定後、AIアシスタントがコンテナにアクセスできるようになります:

```
ユーザー: "myapp-apiコンテナのログを確認して"
Claude: [DockMCPを使用] "ログに500エラーが見えます..."

ユーザー: "APIコンテナでテストを実行して"
Claude: [DockMCPを使用] "npm testを実行中... 3つのテストが通過"
```

## CLIコマンド

DockMCPは2種類のCLIコマンドを提供します:

### ホストOSコマンド（直接Dockerアクセス）

Dockerソケットに直接アクセスするため、**ホストOS上**で実行します:

```bash
# アクセス可能なコンテナを一覧表示
dkmcp list

# コンテナログを取得
dkmcp logs myapp-api --tail 100

# ホワイトリスト登録されたコマンドを実行
dkmcp exec myapp-api "npm test"

# コンテナ詳細をサマリー付きで表示（デフォルト）
dkmcp inspect myapp-api

# コンテナ詳細をフルJSON出力で表示
dkmcp inspect myapp-api --json

# コンテナ統計を取得
dkmcp stats myapp-api
```

**`list` 出力例：**
```
NAME              ID            IMAGE           STATE    STATUS          PORTS
myapp-api         4a2e541171d9  node:18-alpine  running  Up 5 minutes    0.0.0.0:3000->3000/tcp
myapp-proxy       8b3f621283e1  nginx:alpine    running  Up 5 minutes    0.0.0.0:80->80/tcp
```

**`inspect` サマリー出力例：**
```
=== Container Summary: myapp-api ===

State:    running
Started:  2024-01-15T10:30:00Z
Image:    node:18-alpine

--- Network ---
  bridge:
    IP:      172.17.0.2
    Gateway: 172.17.0.1

--- Ports ---
  0.0.0.0:3000 -> 3000/tcp

--- Mounts ---
  /path/to/workspace -> /app (rw)

--- Full Details (JSON) ---
{ ... }
```

### クライアントコマンド（HTTP API経由）

DockMCPサーバーにHTTP経由で接続するため、**DevContainer内**でも使用できます:

```bash
# DockMCPサーバー経由でコンテナを一覧表示
dkmcp client list

# サーバー経由でコンテナログを取得
dkmcp client logs securenote-api --tail 100

# サーバー経由でコンテナ詳細を表示（デフォルトはサマリー）
dkmcp client inspect securenote-api

# サーバー経由でコンテナ詳細を表示（フルJSON）
dkmcp client inspect securenote-api --json

# サーバー経由でコンテナ統計を取得
dkmcp client stats securenote-api

# サーバー経由でコマンドを実行
dkmcp client exec securenote-api "npm test"

# カスタムサーバーURL
dkmcp client list --url http://localhost:8080

# または環境変数を使用
export DOCKMCP_SERVER_URL=http://host.docker.internal:8080
dkmcp client list
```

**どちらを使うべきか:**
- **ホストOSコマンド**: Dockerソケットへの直接アクセスがある場合
- **クライアントコマンド**: DevContainer内、またはDockerソケットアクセスがない環境

## セキュリティモード

### Strictモード
- 読み取り専用操作（logs、inspect、stats）
- コマンド実行不可
- 最も制限が厳しく安全

### Moderateモード（推奨）
- 読み取り操作許可
- コマンド実行はホワイトリストに限定
- 安全性と機能性のバランスが良い

### Permissiveモード
- すべての操作許可
- 信頼された開発環境でのみ使用

## 認証

現在のバージョンでは認証機能は**実装されていません**。

DockMCPはローカル開発環境での使用を想定しており、サーバーはデフォルトでlocalhostにバインドされます。

**将来の計画:**
- 設定ファイルによるオプション認証
- リモートアクセス用のトークンベース認証
- ユーザーの需要に応じて実装予定

認証機能が必要な場合は、[Discussions](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/discussions)でリクエストしてください。

## 設定リファレンス

完全な設定オプションについては [configs/dkmcp.example.yaml](configs/dkmcp.example.yaml) を参照してください:
- コンテナホワイトリストパターン
- コンテナごとのコマンドホワイトリスト
- 監査ログ
- ポートとホスト設定

### ファイルアクセスブロック（blocked_paths）


#### 設定例

```yaml
security:
  blocked_paths:
    # 手動でブロックするパス
    manual:
      "securenote-api":
        - "/.env"
        - "/secrets/*"
      "*":  # 全コンテナに適用
        - "*.key"
        - "*.pem"

    # DevContainer設定からの自動インポート
    auto_import:
      enabled: true
      workspace_root: "."

      # スキャンするファイル
      scan_files:
        - ".devcontainer/docker-compose.yml"
        - ".devcontainer/devcontainer.json"

      # グローバルパターン（全コンテナに適用）
      global_patterns:
        - ".env"
        - "*.key"
        - "secrets/*"

      # Claude Code設定からのインポート
      claude_code_settings:
        enabled: true
        max_depth: 1  # サブディレクトリをスキャンする深度
        settings_files:
          - ".claude/settings.json"
          - ".claude/settings.local.json"
```

#### max_depth の動作

`max_depth` はClaude Code設定ファイルをスキャンする深度を制御します：

```
/workspace/                          ← dkmcp serve 起動位置
├── .claude/settings.json            ← ✅ 見る (depth 0)
├── demo-apps/
│   └── .claude/settings.json        ← ✅ 見る (depth 1)
├── demo-apps-ios/
│   └── .claude/settings.json        ← ✅ 見る (depth 1)
└── demo-apps/subproject/
    └── .claude/settings.json        ← ❌ 見ない (depth 2)
```

| max_depth | 動作 |
|-----------|------|
| 0 | workspace_root のみ |
| 1 | 1階層下まで |
| 2 | 2階層下まで |

#### Claude Code設定との連携

Claude Codeの `.claude/settings.json` にある `permissions.deny` パターンを自動的にインポートできます：

```json
{
  "permissions": {
    "deny": [
      "Read(securenote-api/.env)",
      "Read(**/*.key)",
      "Read(**/secrets/**)"
    ]
  }
}
```

これにより、DevContainerでのClaude Code設定とDockMCPのブロックポリシーを統一できます。

#### ブロック時のレスポンス

アクセスがブロックされると、詳細な理由が返されます：

```json
{
  "blocked": true,
  "reason": "claude_code_settings_deny",
  "pattern": "**/*.key",
  "source": "demo-apps/.claude/settings.json",
  "hint": "This path is blocked by Claude Code settings (permissions.deny)..."
}
```

### 出力マスキング

DockMCPはツール出力内の機密データ（パスワード、APIキー、トークン）をAIアシスタントに返す前に自動的にマスクします。

```yaml
security:
  output_masking:
    enabled: true
    replacement: "[MASKED]"

    # 機密データを検出する正規表現パターン
    patterns:
      - '(?i)(password|passwd|pwd)\s*[=:]\s*["'']?[^\s"''\n]+["'']?'
      - '(?i)(api[_-]?key|secret[_-]?key)\s*[=:]\s*["'']?[^\s"''\n]+["'']?'
      - '(?i)bearer\s+[a-zA-Z0-9._-]+'
      - 'sk-[a-zA-Z0-9]{20,}'
      - '(?i)(postgres|mysql|mongodb|redis)://[^:]+:[^@]+@'

    # マスクする出力
    apply_to:
      logs: true      # get_logs, search_logs
      exec: true      # exec_command
      inspect: true   # inspect_container（環境変数）
```

**例：**
```
# 生の出力
DATABASE_URL=postgres://admin:secret123@db:5432/app

# マスク後
DATABASE_URL=[MASKED]db:5432/app
```

### ホストパスマスキング

ホストOSのパスにユーザーのホームディレクトリが含まれる場合、ホームディレクトリ部分をマスクしてAIから見えないようにします。

```yaml
security:
  host_path_masking:
    enabled: true           # パスマスキングを有効化（デフォルト: true）
    replacement: "[HOST_PATH]"
```

**対応パス：**
- macOS: `/Users/username/...` → `[HOST_PATH]/...`
- Linux: `/home/username/...` → `[HOST_PATH]/...`
- Windows: `C:\Users\username\...` → `[HOST_PATH]\...`

**例（inspect_container出力）：**
```json
// 生の出力
{"Source": "/Users/john/workspace/myapp/.env"}

// マスク後
{"Source": "[HOST_PATH]/workspace/myapp/.env"}
```

> **注意:** このマスキングはMCPツール出力にのみ適用されます。CLIコマンド（`dkmcp inspect`）はユーザー向けにフルパスを表示します。

### パーミッション

グローバルに許可する操作を制御：

```yaml
security:
  permissions:
    logs: true      # ログ取得を許可（get_logs, search_logs）
    inspect: true   # コンテナ検査を許可
    stats: true     # リソース統計を許可
    exec: true      # exec実行を許可（exec_whitelistの対象）
```

### デフォルトコマンド（exec_whitelist `"*"`）

`"*"` をコンテナ名として使用すると、全コンテナで利用可能なコマンドを定義できます：

```yaml
security:
  exec_whitelist:
    # コンテナ固有のコマンド
    "myapp-api":
      - "npm test"
      - "npm run lint"

    # 全コンテナのデフォルトコマンド
    "*":
      - "pwd"
      - "whoami"
      - "date"
```

> ⚠️ **セキュリティ警告:** `env`、`printenv`、`echo *` をデフォルトホワイトリストに追加しないでください。これらは秘匿情報を含む全ての環境変数を露出させる可能性があります。

### 危険モード（exec_dangerously）

ホワイトリストにない `tail`、`grep`、`cat` などのコマンドがデバッグに必要な場合、DockMCPは `blocked_paths` の制限を維持しながらこれらのコマンドを許可する「危険モード」を提供します。

#### なぜ危険モードが必要か？

Dockerの `get_logs` はstdout/stderrのみを表示します。`/var/log/app.log` のようなログファイルを見るには `tail` や `cat` などが必要です。しかし、これらを `exec_whitelist` に追加すると、秘匿情報を含む任意のファイルを読めてしまいます。

危険モードはこれを解決します：
1. 特定のコマンドを許可（例：`tail`、`cat`、`grep`）
2. ファイルパスは引き続き `blocked_paths` でチェック
3. パイプ（`|`）、リダイレクト（`>`）、パストラバーサル（`..`）はブロック

#### 設定

```yaml
security:
  exec_dangerously:
    enabled: false  # グローバル有効/無効
    commands:
      # コンテナ固有のコマンド
      "securenote-api":
        - "tail"
        - "head"
        - "cat"
        - "grep"
      # グローバルコマンド（全コンテナ）
      "*":
        - "tail"
        - "ls"
```

#### サーバー起動フラグ

設定ファイルを変更せずに起動時に危険モードを有効化：

```bash
# 特定のコンテナで有効化
dkmcp serve --dangerously=securenote-api,demo-app

# 全コンテナで有効化
dkmcp serve --dangerously-all
```

これらのフラグは：
- `exec_dangerously.enabled = true` を設定
- デフォルトコマンドを追加：`tail`、`head`、`cat`、`grep`、`less`、`wc`、`ls`、`find`

| フラグ | 動作 |
|-------|------|
| `--dangerously=container1,container2` | 既存の `exec_dangerously.commands` 設定を**クリア**し、指定コンテナのみ有効化 |
| `--dangerously-all` | 既存の設定と**マージ**し、`"*"`（全コンテナ）にコマンドを追加 |

> 💡 特定のコンテナのみに危険モードを厳密に制限したい場合は `--dangerously=container` を使用します。設定ファイルのコンテナ固有の設定を保持しながら広範囲に有効化したい場合は `--dangerously-all` を使用します。

#### 使用方法

**MCPツール（Claude Code）：**
```json
{
  "tool": "exec_command",
  "arguments": {
    "container": "securenote-api",
    "command": "tail -100 /var/log/app.log",
    "dangerously": true
  }
}
```

**CLI：**
```bash
# 直接（ホストOS）
dkmcp exec --dangerously securenote-api "tail -100 /var/log/app.log"

# クライアント（DevContainer）
dkmcp client exec --dangerously --url http://host.docker.internal:8080 securenote-api "tail -100 /var/log/app.log"
```

#### セキュリティモデル

```
dangerously=true でリクエスト
    ↓
1. exec_dangerously.enabled = true か？（サーバー設定）
    ↓
2. ベースコマンドが exec_dangerously.commands にあるか？
    ↓
3. パイプ（|）、リダイレクト（>）、パストラバーサル（..）チェック
    ↓
4. コマンドからファイルパスを抽出
    ↓
5. 各パスを blocked_paths と照合
    ↓
全チェック通過で実行
```

**設計上ブロックされる例：**
- `cat /secrets/key.pem` → `blocked_paths` でブロック
- `cat /etc/passwd | grep root` → パイプ禁止
- `cat ../secrets/key` → パストラバーサル禁止
- `rm /var/log/app.log` → `rm` は `exec_dangerously.commands` にない

> ⚠️ **セキュリティ注意:** クライアントは明示的に `dangerously=true` を設定する必要があります。この「オプトイン」設計により、危険モード使用時の意識的な承認が保証されます。

#### エラーのヒントメッセージ

ホワイトリストにないが危険モードで使用可能なコマンドを実行しようとすると、ヒント付きのエラーが表示されます：

```
command not whitelisted: tail (hint: this command is available with dangerously=true)
```

#### 利用可能なコマンドの確認

`dkmcp client commands` を使用して、ホワイトリストコマンドと危険コマンドの両方を確認できます：

```bash
$ dkmcp client commands
CONTAINER           ALLOWED COMMANDS
---------           ----------------
* (all containers)  pwd
                    whoami
securenote-api      npm test
                    npm run lint

CONTAINER           DANGEROUS COMMANDS (requires dangerously=true)
---------           ----------------------------------------------
* (all containers)  tail
                    ls
securenote-api      tail
                    cat
                    grep

Note: Commands with '*' wildcard match any suffix. Dangerous commands require dangerously=true parameter.
```

## アーキテクチャ

```
┌─────────────────────────────────┐
│ ホストOS                         │
│  ┌──────────────────────────┐   │
│  │ DockMCP (Goバイナリ)    │   │
│  │ - MCPサーバー(HTTP/SSE)  │   │
│  │ - セキュリティポリシー    │   │
│  └────────┬─────────────────┘   │
│           │ :8080                │
│  ┌────────┴─────────────────┐   │
│  │ Docker Engine            │   │
│  │  ├─ DevContainer         │   │
│  │  │   └─ Claude Code ─┐   │   │
│  │  ├─ app-api ←─────────┘   │   │
│  │  └─ app-db              │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

## 設計思想

**なぜDockMCPは `docker-compose up/down` やコンテナ再起動をサポートしないのか？**

これは意図的な設計です。DockMCPは明確な責任分離を念頭に設計されています。

### 基本設計原則

```
AI = 目と口（観察する、提案する）
人間 = 手（実行する）
```

**AIができること：**
- ログ、統計情報、コンテナ情報の読み取り
- ホワイトリストに登録されたコマンドの実行（テスト、リント）
- ファイルの読み取り（blocked_paths による保護付き）
- 変更や解決策の提案

**人間がやること：**
- コンテナの起動/停止
- イメージのリビルド
- インフラの変更
- 破壊的な操作の実行

### なぜこの分離が必要なのか？

#### 1. Dockerfile の変更にはリビルドが必要

Dockerfile を変更した場合、単純な `restart` では変更が反映されません：

```bash
# これでは Dockerfile の変更は反映されない
docker restart myapp  # 古いイメージのまま

# 実際に必要なのはこれ
docker-compose build myapp
docker-compose up -d myapp
```

DockMCP が `restart` だけをサポートしても、実際の問題を解決せずに誤った安心感を与えることになります。

#### 2. ほとんどの開発作業にコンテナ操作は不要

| アクション | 対応方法 | コンテナ操作が必要？ |
|-----------|---------|-------------------|
| コード変更 | ホットリロード / `exec npm run dev` | 不要 |
| 設定ファイル変更 | アプリ再読み込みコマンド | 不要 |
| テスト実行 | `exec npm test` | 不要 |
| ログ確認 | `get_logs` | 不要 |
| Dockerfile 変更 | リビルド + 再作成 | **必要、ただし人間が実行** |

コンテナ操作が本当に必要なケース（Dockerfile変更、docker-compose.yml変更、環境変数変更）は**インフラの変更**であり、人間のレビューを経るべきです。

#### 3. リスク vs 頻度のトレードオフ

| 操作レベル | リスク | 開発中の頻度 |
|-----------|-------|-------------|
| ログ/統計の読み取り | なし | 非常に高い |
| ホワイトリストコマンド実行 | 低 | 高い |
| コンテナ再起動 | 中 | 低い |
| ビルド/再作成 | 高 | 非常に低い |

低頻度のユースケースのために高リスクな操作を追加するのは合理的ではありません。

#### 4. AIは調査し、人間が行動する

**良いワークフロー：**
1. AIがログ、統計、エラーパターンを調査
2. AIが問題を特定し、解決策を提案
3. **人間が** 再起動/リビルドするかどうかを決定・実行

**リスクのあるワークフロー：**
1. AIがエラーを検知して即座にコンテナを再起動
2. 問題が解決しない（または悪化する）
3. 人間が何が起きたか理解できない

### exec_command について

`exec_command` はホワイトリストで許可するコマンドを絞り込めます：

```yaml
exec_whitelist:
  "myapp-api":
    - "npm test"
    - "npm run lint"
    - "npm run dev"  # 開発サーバーの再起動が可能
```

これにより可能になること：
- テストとリントの実行
- 開発サーバーの再起動（プロセスマネージャー経由）
- ヘルスチェックとデバッグコマンド

許可されないこと：
- 任意のコマンド実行
- ファイルシステムの変更
- ネットワーク設定の変更

### まとめ

DockMCP の現在の設計：
- コンテナ情報への**読み取り専用アクセス**（ログ、統計、inspect）
- ホワイトリストによる**制御されたコマンド実行**
- blocked_paths 保護付きの**ファイルアクセス**
- **コンテナライフサイクル操作なし**（start/stop/restart/build）

これにより、AIが影響を与えられる範囲に明確な境界を維持し、AIによる開発支援の恩恵を受けることができます。

## 提供されるMCPツール

| ツール | 説明 |
|------|------|
| `list_containers` | アクセス可能なコンテナを一覧表示 |
| `get_logs` | コンテナログを取得 |
| `get_stats` | リソース使用統計を取得 |
| `exec_command` | ホワイトリスト登録されたコマンドを実行（`dangerously`モード対応） |
| `inspect_container` | 詳細なコンテナ情報を取得 |
| `get_allowed_commands` | コンテナごとのホワイトリストコマンドを一覧表示 |
| `get_security_policy` | 現在のセキュリティ設定を表示 |
| `search_logs` | パターンマッチでコンテナログを検索 |
| `list_files` | コンテナ内のディレクトリをリスト表示（ブロック機能付き） |
| `read_file` | コンテナ内のファイルを読み取り（ブロック機能付き） |
| `get_blocked_paths` | ブロックされているファイルパスを表示 |

## トラブルシューティング

### DockMCPサーバーが認識されない

1. **ホストでDockMCPが実行されていることを確認：**
   ```bash
   curl http://localhost:8080/health
   # 200 OK を返すはず
   ```

2. **DevContainer内でMCP設定を確認：**
   ```bash
   cat ~/.claude.json | jq '.mcpServers.dkmcp'
   # "url": "http://host.docker.internal:8080/sse" であること
   ```

3. **MCP再接続を試す：**
   Claude Codeで `/mcp` → 「Reconnect」を選択

4. **VS Codeを完全に再起動：**
   macOS: `Cmd + Q` / Windows・Linux: `Alt + F4`

### DockMCPサーバーを再起動した場合

DockMCPサーバーを再起動するとSSE接続が切断されるため、AIアシスタント側で再接続が必要です：

- **Claude Code:** `/mcp` → 「Reconnect」を選択
- **それでも解決しない場合:** VS Codeを完全に再起動（Cmd+Q / Alt+F4）

### "Connection refused" エラー

- DockMCPがホストで動作しているか？ → `ps aux | grep dkmcp`
- URLに `host.docker.internal` を使っているか？（`localhost` はNG）
- ポート8080がファイアウォールでブロックされていないか？ → `lsof -i :8080`

### "Container not in allowed list"

設定の `allowed_containers` にコンテナ名またはパターンを追加:
```yaml
security:
  allowed_containers:
    - "your-container-name"
```

### "Command not whitelisted"

設定の `exec_whitelist` にコマンドを追加:
```yaml
security:
  exec_whitelist:
    "container-name":
      - "your command here"
```

## ライセンス

MIT License - [LICENSE](LICENSE) ファイルを参照

## 謝辞

- [Model Context Protocol](https://modelcontextprotocol.io/) で構築
- Docker統合は [docker/docker](https://github.com/docker/docker) 経由
- CLIは [spf13/cobra](https://github.com/spf13/cobra) で駆動

---

**注意**: DockMCPは制御されたアクセスを提供しますが、責任を持って使用してください。AIアシスタントに公開する前に、必ずセキュリティ設定を確認してください。
