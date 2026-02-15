# DockMCP ホストアクセス

DockMCP はコンテナだけでなく、ホスト OS への操作も制御付きで AI に提供できます。**ホストツール**、**コンテナライフサイクル**、**ホストコマンド** の 3 機能があり、いずれも監査可能な形でホスト操作を実行します。

[<- README に戻る](../README.ja.md)

---

## 目次

- [全体像](#全体像)
- [ホストツール](#ホストツール)
- [コンテナライフサイクル](#コンテナライフサイクル)
- [ホストコマンド](#ホストコマンド)
- [MCP ツールリファレンス](#mcp-ツールリファレンス)
- [CLI コマンドリファレンス](#cli-コマンドリファレンス)
- [セキュリティ上の注意点](#セキュリティ上の注意点)

---

## 全体像

```
AI Sandbox（コンテナ）
  │
  │  MCP / HTTP
  ▼
DockMCP サーバー（ホスト OS）
  ├── コンテナアクセス           ← 既存（ログ、exec、統計 等）
  ├── ホストツール               ← 新機能: 承認済みスクリプトをホストで実行
  ├── コンテナライフサイクル     ← 新機能: コンテナの起動/停止/再起動
  └── ホストコマンド             ← 新機能: ホワイトリスト登録された CLI コマンドをホストで実行
```

ホストツールはデフォルトで有効ですが、コンテナライフサイクルとホストコマンドはデフォルトで無効です。いずれも設定フィアルで有効にできます。

---

## ホストツール

### 概要

`.sandbox/host-tools/` に配置されたスクリプト（`.sh`、`.go`、`.py` 等）を DockMCPを通して AI が実行できる機能です。
ここに配置されたスクリプトをそのまま実行するのではなく、DockMCPサーバー起動時の対話型の入力で承認する事により、コンテナからアクセスのできないホストOS側に配置され、DockMCPを通じてホスト側の操作（デモコンテナの起動など）を行える様になります。

これは、コンテナ内からホスト上で実行されるソースコードをAIが勝手に修正できない様にするための仕組みです。


### 承認ワークフロー

ツールは 2 段階のプロセスを経て実行可能になります。

```
1. .sandbox/host-tools/ にスクリプトを配置     （ステージング — ワークスペース内）
2. ホスト OS で実行: dkmcp tools sync          （レビュー＆承認）
3. 承認済みコピーが ~/.dkmcp/host-tools/<project-id>/ に保存
4. AI は承認済みバージョンのみをDockMCPを通して実行可能
```

実行されるのは承認済みのコピーだけです。ステージング側が変更されると `dkmcp tools sync` が差分を検出し、再承認を求めます。

### ディレクトリ構成

```
~/.dkmcp/host-tools/
├── _common/                    # 全プロジェクト共通
│   └── shared-tool.sh
└── <project-id>/               # プロジェクト固有の承認済みツール
    ├── .project                # プロジェクトメタデータ（ワークスペースパス等）
    └── demo-build.sh           # 承認済みツール
```

- **`_common/`** — 全プロジェクトで共有されるツール（`common: true` で有効化）
- **`<project-id>/`** — ワークスペースパスから自動生成され、プロジェクトごとにツールを隔離

### 設定

```yaml
# dkmcp.yaml
host_access:
  host_tools:
    enabled: true
    approved_dir: "~/.dkmcp/host-tools"
    staging_dirs:
      - ".sandbox/host-tools"
    common: true
    allowed_extensions: [".sh", ".go", ".py"]
    timeout: 60  # 秒
```

### 付属のデモツール

| ツール | 説明 |
|--------|------|
| `demo-build.sh` | デモアプリの Docker イメージをビルド |
| `demo-up.sh` | デモコンテナを起動（`docker-compose up -d`） |
| `demo-down.sh` | デモコンテナを停止（`docker-compose down`） |
| `copy-credentials.sh` | DevContainer プロジェクト間でホームディレクトリをコピー |

### 自作ホストツールの書き方

`.sandbox/host-tools/` にスクリプトを配置し、先頭に説明ヘッダーを記述します。

```bash
#!/bin/bash
# my-tool.sh
# このツールの簡単な説明
#
# Usage:
#   my-tool.sh [options] <args>
#
# Examples:
#   my-tool.sh --verbose build
```

ヘッダーは DockMCP がパースし、`list_host_tools` や `get_host_tool_info` で AI に提供されます。

---

## コンテナライフサイクル

### 概要

Docker API を直接使って、コンテナの起動・停止・再起動を AI が行える機能です。クラッシュしたコンテナの復旧や、設定変更の反映に便利です。

### 仕組み

- **Docker API** を直接呼び出します（`docker` CLI ではありません）
- `lifecycle: true` を設定ファイルで有効化する必要があります
- `allowed_containers` ポリシーに従い、許可されたコンテナだけを操作可能
- タイムアウトパラメータで graceful shutdown を制御可能

### 設定

```yaml
# dkmcp.yaml
security:
  permissions:
    lifecycle: true  # デフォルト: false
```

### 使い方

AI は MCP ツール（`restart_container`、`stop_container`、`start_container`）または CLI フォールバックを使います。

```bash
# AI Sandbox から実行
dkmcp client restart securenote-api
dkmcp client stop securenote-api --timeout 30
dkmcp client start securenote-api
```

---

## ホストコマンド

### 概要

ホスト OS 上でホワイトリスト登録された CLI コマンドを AI が実行できる機能です。git の状態確認やディスク使用量のチェックなど、フルシェルアクセスを与えずに必要な操作だけを許可します。

### 仕組み

コマンドは 3 層で制御されます。

1. **ホワイトリスト** — ベースコマンド＋引数パターンのマッチングが必要
2. **拒否リスト** — ホワイトリストを上書きして特定の危険な組み合わせをブロック
3. **危険モード** — `dangerously=true` を明示的に指定する必要がある別枠のコマンドセット

### 設定

```yaml
# dkmcp.yaml
host_access:
  host_commands:
    enabled: true

    # ホワイトリスト（ベースコマンド → 許可する引数パターン）
    whitelist:
      "git":
        - "status"            # 完全一致
        - "diff *"            # プレフィックスマッチ: diff HEAD, diff --stat 等
        - "log --oneline *"   # プレフィックスマッチ
      "df":
        - "-h"
      "free":
        - "-m"

    # 拒否リスト（ホワイトリストを上書き）
    # deny:
    #   "git":
    #     - "push --force *"

    # 危険モード（dangerously=true が必要）
    dangerously:
      enabled: false
      commands:
        "git":
          - "checkout"
          - "pull"
```

### 引数パターンのマッチング

| パターン | マッチする | マッチしない |
|----------|-----------|-------------|
| `"status"` | `git status` | `git status --short` |
| `"diff *"` | `git diff`、`git diff HEAD`、`git diff --stat` | — |
| `"-h"` | `df -h` | `df -h /tmp` |

### 組み込みの安全機構

ホワイトリストの設定にかかわらず、以下は**常にブロック**されます。

- **パイプ**（`|`）— コマンドの連結を防止
- **リダイレクト**（`>`、`<`）— ファイル操作を防止
- **パストラバーサル**（`..`）— ワークスペース外への脱出を防止
- **ブロックパス** — ファイル引数は、コンテナファイルアクセスと同じ `blocked_paths` ポリシーで検証

---

## MCP ツールリファレンス

| MCP ツール | 説明 | 機能 |
|------------|------|------|
| `list_host_tools` | ホストツール一覧と説明を表示 | ホストツール |
| `get_host_tool_info` | ツールの使い方・実行例を表示 | ホストツール |
| `run_host_tool` | 承認済みホストツールを実行 | ホストツール |
| `restart_container` | コンテナを再起動（Docker API） | ライフサイクル |
| `stop_container` | コンテナを停止（Docker API） | ライフサイクル |
| `start_container` | コンテナを起動（Docker API） | ライフサイクル |
| `exec_host_command` | ホワイトリスト登録されたホストコマンドを実行 | ホストコマンド |

---

## CLI コマンドリファレンス

### ホストツール管理（ホスト OS で実行）

```bash
# ステージングディレクトリからツールをレビュー・承認
dkmcp tools sync

# 承認済みツールのディレクトリとプロジェクト情報を表示
dkmcp tools list
```

### ホストツールクライアント（AI Sandbox から実行）

```bash
dkmcp client host-tools list
dkmcp client host-tools info <ツール名>
dkmcp client host-tools run <ツール名> [引数...]
```

### コンテナライフサイクル（AI Sandbox から実行）

```bash
dkmcp client restart <コンテナ名> [--timeout <秒>]
dkmcp client stop <コンテナ名> [--timeout <秒>]
dkmcp client start <コンテナ名>
```

### ホストコマンド（AI Sandbox から実行）

```bash
dkmcp client host-exec "git status"
dkmcp client host-exec --dangerously "git pull"
```

---

## セキュリティ上の注意点

### ホストツール

- **承認が必須** — 実行前にかならず承認が必要です。ステージングディレクトリ（ワークスペース内）は AI が書き込めますが、承認済みディレクトリ（`~/.dkmcp/host-tools/`）は書き込めません。
- **変更検出** — SHA256 ハッシュで変更を検出します。変更されたツールは再承認が必要です。
- **タイムアウト** — ツールの実行にはタイムアウトがあり（デフォルト: 60 秒）、暴走スクリプトを防止します。
- **拡張子の制限** — `.sh`、`.go`、`.py` のみがツールとして登録可能です。

### コンテナライフサイクル

- **オプトイン** — デフォルトは無効（`lifecycle: false`）です。
- **コンテナスコープ** — `allowed_containers` ポリシーに従います。
- **Docker API のみ** — シェル実行ではなく Docker API を直接使います。

### ホストコマンド

- **ホワイトリスト限定** — 明示的にリストされたベースコマンド＋引数パターンのみ許可されます。
- **拒否リストが優先** — 拒否リストはホワイトリストに勝ちます。
- **パイプ・リダイレクト禁止** — `|`、`>`、`<` は設定にかかわらず常にブロックされます。
- **パストラバーサル禁止** — `..` は常にブロックされます。
- **ブロックパス** — ファイルパスの引数は、コンテナファイルアクセスと同じ `blocked_paths` ポリシーで検証されます。
- **危険モード** — 慎重に扱うべきコマンド（例: `git pull`、`git checkout`）は `dangerously` セクションに分離でき、呼び出し側が明示的に `dangerously=true` を指定する必要があります。

### 共通

- **監査ログ** — 監査ログを有効にすると、すべてのホストアクセス操作が記録されます。
- **出力マスキング** — ツールやコマンドの出力に含まれる機密データは、AI に返す前にマスクされます。
- **ホストパスマスキング** — ホスト OS のパス（例: `/Users/username/`）はマスクされ、AI がホストユーザーの身元を知ることを防ぎます。
