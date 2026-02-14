# アーキテクチャ詳細

AI Sandbox + DockMCP の仕組みを図解で詳しく説明します。

[← README に戻る](../README.ja.md)

---

## 全体構成

```
┌───────────────────────────────────────────────────┐
│ Host OS                                           │
│                                                   │
│  ┌──────────────────────────────────────────────┐ │
│  │ DockMCP Server                               │ │
│  │  HTTP/SSE API for AI                         ←─────┐
│  │  Security policy enforcement                 │ │   │
│  │  Container access gateway                    │ │   │
│  │                                              │ │   │
│  └────────────────────↑─────────────────────────┘ │   │
│                       │ :8080                     │   │
│  ┌────────────────────│─────────────────────────┐ │   │
│  │ Docker Engine      │                         │ │   │
│  │                    │                         │ │   │
│  │   AI Sandbox  ←────┘                         │ │   │
│  │    ├─ Claude Code / Gemini                   │ │   │
│  │    ├─ SandboxMCP (stdio)                     │ │   │
│  │    └─ secrets/ → empty (hidden)              │ │   │
│  │                                              │ │   │
│  │   API Container    ←───────────────────────────────┘
│  │    └─ secrets/ → real files                  │ │   │
│  │                                              │ │   │
│  │   Web Container    ←───────────────────────────────┘
│  │                                              │ │
│  └──────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────┘
```

<details>
<summary>ツリー形式で見る</summary>

**データフロー:** AI (AI Sandbox) → DockMCP (:8080) → 他のコンテナ

```
ホストOS
├── DockMCP Server (:8080)
│   ├── AI用 HTTP/SSE API
│   ├── セキュリティポリシー実施
│   └── コンテナアクセスゲートウェイ
│
└── Docker Engine
    ├── AI Sandbox (AI環境)
    │   ├── Claude Code / Gemini
    │   ├── SandboxMCP (stdio)
    │   └── secrets/ → 空（隠蔽）
    │
    ├── API Container
    │   └── secrets/ → 実ファイル
    │
    └── Web Container
```

</details>

---

## 秘匿ファイルの隠蔽の仕組み

AIがAI Sandbox内で動作するため、Dockerボリュームマウントで秘匿ファイルを隠せます。

```
ホストOS
├── demo-apps/securenote-api/.env  ← 実体あり
│
├── AI Sandbox (AI実行環境)
│   └── AI が .env を読もうとする
│       → /dev/null にマウントされているため空に見える
│
└── API Container (実行環境)
    └── Node.js アプリが .env を読む
        → 正常に読める
```

**結果:**
- AIは秘匿ファイルを読めない（セキュリティ確保）
- アプリは秘匿ファイルを読める（機能は維持）
- DockMCP経由でAIがログ確認・テスト実行は可能

---

## AI Sandboxによる隔離のメリット

AIがAI Sandbox内で動作することで、ホストOSのファイルへのアクセスも制限されます。

```
ホストOS
├── /etc/            ← AIからアクセス不可
├── ~/.ssh/          ← AIからアクセス不可
├── ~/Documents/     ← AIからアクセス不可
├── ~/other-project/ ← AIからアクセス不可
├── ~/secret-memo/   ← AIからアクセス不可
│
└── AI Sandbox
    └── /workspace/   ← ここだけ見える
        ├── demo-apps/
        ├── dkmcp/
        └── ...
```

**メリット:**
- ホストOSのシステムファイルに触れない
- 他のプロジェクトに触れない
- SSH鍵や認証情報（`~/.ssh/`）に触れない
- ホストOSを誤って変更するリスクがない

---

## セキュリティ機能の詳細

### 1. 秘匿情報の隠蔽

Dockerボリュームマウントを使ってAIから秘匿情報を隠蔽：

```yaml
# .devcontainer/docker-compose.yml
volumes:
  # 秘匿ファイルを隠す
  - /dev/null:/workspace/demo-apps/securenote-api/.env:ro

tmpfs:
  # 秘匿ディレクトリを隠す
  - /workspace/demo-apps/securenote-api/secrets:ro
```

**結果:**
- AIには空のファイル/ディレクトリに見える
- 実際のコンテナは本物の秘匿情報にアクセス可能
- 開発は通常通り動作！

**実際の動作例:**

```bash
# AI Sandbox内から（AIが試しても失敗する）
$ cat demo-apps/securenote-api/secrets/jwt-secret.key
(空)

# しかしClaude Codeに聞いてみる:
"APIが秘匿情報にアクセスできているか確認して"

# Claude は DockMCP を使ってクエリ:
$ curl http://localhost:8080/api/demo/secrets-status

# レスポンスでAPIが秘匿情報を持っていることが証明される:
{
  "secretsLoaded": true,
  "proof": {
    "jwtSecretLoaded": true,
    "jwtSecretPreview": "super-sec***"
  }
}
```

### 2. 制御されたコンテナアクセス

DockMCPがセキュリティポリシーを実施：

```yaml
# dkmcp.yaml
security:
  mode: "moderate"  # strict | moderate | permissive

  allowed_containers:
    - "demo-*"
    - "project_*"

  exec_whitelist:
    "securenote-api":
      - "npm test"
      - "npm run lint"
```

コンテナ内の機密ファイルブロック（`blocked_paths`）、Claude Code / Gemini 設定からの自動インポートなど、詳細な設定は [dkmcp/README.ja.md「設定リファレンス」](../dkmcp/README.ja.md#設定リファレンス) を参照。

**実際の動作例 — クロスコンテナでのデバッグ:**

```bash
# バグをシミュレート: Webアプリでログインできない

# Claude Codeに聞く:
"ログインが失敗しています。APIのログを確認してもらえますか？"

# ClaudeはDockMCPを使ってログを取得:
dkmcp.get_logs("securenote-api", { tail: "50" })

# ログからエラーを発見:
"JWT verification failed - invalid secret"

# Claude Codeに聞く:
"APIのテストを実行して確認してください"

# Claude は DockMCP経由でテストを実行:
dkmcp.exec_command("securenote-api", "npm test")

# 問題が特定され、修正完了！
```

### 3. 基本的なサンドボックス保護

- **非rootユーザー**: `node` ユーザーとして実行
- **制限されたsudo**: パッケージマネージャーのみ（apt、npm、pip）
- **認証情報の永続化**: `.claude/`、`.config/gcloud/` 用の名前付きボリューム

> ⚠️ **セキュリティ注意: npm/pip3 sudoのリスク**
>
> npm/pip3にsudoを許可すると、悪意のあるパッケージを通じて悪用される可能性があります。悪意のあるpostinstallスクリプトは、昇格された権限で任意のコードを実行できます。
>
> **対策オプション:**
> 1. npm/pip3をsudoersから削除（`.sandbox/Dockerfile`を編集）
> 2. `npm install --ignore-scripts` フラグを使用
> 3. 必要なパッケージをDockerfileで事前インストール
> 4. `.npmrc`に`ignore-scripts=true`を設定

### 4. 出力マスキング（多層防御）

万が一シークレットがログやコマンド出力に含まれていても、DockMCPが自動的にマスクします：

```
# 生のログ出力
DATABASE_URL=postgres://user:secret123@db:5432/app

# AIが見る内容（マスク後）
DATABASE_URL=[MASKED]db:5432/app
```

パスワード、APIキー、Bearerトークン、認証情報付きデータベースURLなどをデフォルトで検出します。設定方法の詳細は [dkmcp/README.ja.md「出力マスキング」](../dkmcp/README.ja.md#出力マスキング) を参照。

### 5. Docker ソケットを渡さない理由

「AI Sandbox に Docker ソケット（`/var/run/docker.sock`）を渡せば、DockMCP なしで直接コンテナ操作できるのでは？」と思うかもしれません。しかし、これは**セキュリティ上やってはいけません**。

Docker ソケットへのアクセスは、ホスト OS の管理者権限を持つのとほぼ同じです。AI にソケットを渡すと：

- `docker exec` で他コンテナの **`.env` や `secrets/` を直接読める**（隠蔽が無意味に）
- `docker run -v /:/host` で**ホスト OS のファイルシステム全体**をマウントできる
- コンテナの停止・削除・イメージの操作が自由にできる

つまり、ボリュームマウントで秘匿ファイルを隠していても、ソケット経由で簡単に回避できてしまいます。

**DockMCP はこの問題を解決するために存在します：**

| | Docker ソケット直接 | DockMCP 経由 |
|---|---|---|
| 秘匿ファイル | 読める | **ブロック** |
| 実行できるコマンド | 制限なし | **ホワイトリストのみ** |
| ログの秘匿情報 | そのまま見える | **自動マスク** |
| コンテナの停止・削除 | できる | **できない** |

DockMCP は「AI に必要な操作（ログ確認、テスト実行など）だけ」を安全に提供するゲートウェイです。

---

## マルチプロジェクトワークスペース

これらのセキュリティ機能により、複数プロジェクトを1つのワークスペースで安全に扱えます。

このデモ環境の例：
- **バックエンドAPI** (demo-apps/securenote-api)
- **Webフロントエンド** (demo-apps/securenote-web)
- **iOSアプリ** (demo-apps-ios/)

AIができること：
- すべてのソースコードを読む（アプリとサーバー間の連携不具合の調査が可能）
- 任意のコンテナのログを確認（DockMCP経由）
- プロジェクト横断でテストを実行
- クロスコンテナの問題をデバッグ
- **秘匿情報には一切触れない**

---

## SandboxMCP - コンテナ内MCPサーバー

DockMCP（ホストOS側）とは別に、**SandboxMCP** がコンテナ内で動作します。

```
┌─────────────────────────────────────────────────────┐
│ AI Sandbox (コンテナ内)                               │
│                                                     │
│  ┌─────────────────┐      ┌─────────────────────┐  │
│  │ Claude Code     │ ←──→ │ SandboxMCP (stdio)  │  │
│  │ Gemini CLI      │      │                     │  │
│  └─────────────────┘      │ • list_scripts      │  │
│                           │ • get_script_info   │  │
│                           │ • run_script        │  │
│  ┌─────────────────────┐  │ • list_tools        │  │
│  │ .sandbox/scripts/   │  │ • get_tool_info     │  │
│  │ • validate-secrets  │←─│ • run_tool          │  │
│  │ • sync-secrets      │  └─────────────────────┘  │
│  │ • help              │                           │
│  │ • ...               │                           │
│  └─────────────────────┘                           │
└─────────────────────────────────────────────────────┘
```

### DockMCP と SandboxMCP の役割分担

| | SandboxMCP | DockMCP |
|---|---|---|
| 動作場所 | コンテナ内 | ホストOS |
| 通信方式 | stdio | SSE (HTTP) |
| 用途 | スクリプト/ツールの発見・実行 | 他コンテナへのアクセス |
| 起動方法 | AI CLIが自動起動 | 手動 (`dkmcp serve`) |

### 6つのMCPツール

| ツール | 説明 | 使用例 |
|--------|------|--------|
| `list_scripts` | スクリプト一覧を表示 | 「使えるスクリプトは？」 |
| `get_script_info` | スクリプトの詳細情報 | 「validate-secrets.sh の使い方は？」 |
| `run_script` | コンテナ内スクリプトを実行 | 「validate-secrets.sh を実行して」 |
| `list_tools` | ツール一覧を表示 | 「使えるツールは？」 |
| `get_tool_info` | ツールの詳細情報 | 「search-history の使い方は？」 |
| `run_tool` | ツールを実行 | 「会話履歴から 'MCP' を検索して」 |

### ホスト専用スクリプトの取り扱い

一部のスクリプト（`init-host-env.sh` など）はホスト OS で実行する必要があるため、コンテナ内では実行できません。

> **注:** `copy-credentials.sh` は `.sandbox/host-tools/` に移動し、DockMCP の `run_host_tool` MCPツール経由で実行できるようになりました。

```
AIが run_script("init-host-env.sh") を呼び出すと:

┌────────────────────────────────────────────────────────────┐
│ ❌ このスクリプト (init-host-env.sh) は                     │
│    ホストOSで実行する必要があります。                        │
│                                                            │
│ ホストマシンで以下を実行してください:                        │
│   .sandbox/scripts/init-host-env.sh                        │
│                                                            │
│ AI Sandbox にはDockerソケットへのアクセス権がないため、      │
│ ホスト専用スクリプトは実行できません。                       │
└────────────────────────────────────────────────────────────┘
```

**結果:** エラーではなく、明確なガイダンスが返される

### 自動登録

SandboxMCP はコンテナ起動時に自動でビルド・登録されます：

- **DevContainer**: `postStartCommand` で実行
- **CLI Sandbox**: 起動スクリプト内で実行
- **Claude Code / Gemini CLI 両対応**: CLIがインストールされていれば登録

手動登録が必要な場合：

```bash
cd /workspace/.sandbox/sandbox-mcp
make register    # ビルドして登録
make unregister  # 登録解除
```

### 自作ツールの追加

Go ファイルを `.sandbox/tools/` に置くだけで、SandboxMCP が自動的に認識します。`package` 宣言前のコメントからメタデータが抽出されます。`// ---` 区切り線があると、そこでパースが停止します（日本語説明等を区切り線の下に書けます）：

```go
// Short description (最初のコメント行が description になります)
//
// Usage:
//   go run .sandbox/tools/my-tool.go [options] <args>
//
// Examples:
//   go run .sandbox/tools/my-tool.go "hello"
//   go run .sandbox/tools/my-tool.go -verbose "world"
//
// --- 以下はパーサー対象外（任意） ---
//
// ツールの日本語説明
package main
```

```
┌───────────────────────────────────────────────────┐
│ .sandbox/tools/                                   │
│  ├── search-history.go   ← 組み込みツール         │
│  └── my-tool.go          ← ファイルを置くだけ     │
│                                                   │
│ SandboxMCP が *.go ファイルを自動検出             │
│ 登録や設定は不要                                  │
└───────────────────────────────────────────────────┘
```

AI アシスタントは `list_tools` でツールを発見し、`get_tool_info` で使い方を確認し、`run_tool` で実行できます。

### 自作スクリプトの追加

`.sandbox/scripts/` にシェルスクリプトを置いても同様に認識されます。シェルスクリプトから Python や Node.js など他の言語を呼び出すこともできるため、Go 以外の言語でもツールを作成できます。

**ヘッダー形式：**

```bash
#!/bin/bash
# my-script.sh
# English description (can be multi-line)
# Additional description continues here
# ---
# 日本語の説明（任意、パースされない）
```

- 1行目: シバン行
- 2行目: ファイル名
- 3行目以降: 英語の説明（複数行可、`list_scripts` で AI に表示される）
- N行目: `# ---` 区切り（パースはここで停止）
- N+1行目以降: 日本語説明など（AI には渡されない）

`# ---` 区切り以降はパーサーが無視しますが、人間が読むための情報として残せます。これは Go ツールの `// ---` パターンと同じ設計です。

**Usage セクション（任意）：**

`# ---` 区切りの前に `Usage:` または `使用法:` で始まるコメント行があると、`get_script_info` で使い方として表示されます。空のコメント行でセクションが終了します。これは Go ツールで Usage/Examples が `// ---` の前に来るパターンと同じです。

```bash
#!/bin/bash
# my-script.sh
# English description
#
# Usage:
#   my-script.sh [options] <args>
#   my-script.sh --verbose "hello"
#
# ---
# 日本語の説明
```

**スキップされるファイル：**

| パターン | 理由 |
|---|---|
| `_` で始まるファイル | ライブラリ扱い（例: `_startup_common.sh`） |
| `help.sh` | ヘルプスクリプト自体は一覧に含めない |
| `.sh` 以外のファイル | 対象外 |

**カテゴリの自動分類：**

| ファイル名 | カテゴリ |
|---|---|
| `test-` で始まる | `test` |
| それ以外 | `utility` |

**実行環境の分類：**

スクリプトは実行環境によって3種類に分類されます。ホスト専用スクリプトを `run_script` で実行しようとすると、ホスト OS での実行コマンドを案内するエラーが返されます。

| 環境 | 対象スクリプト |
|---|---|
| `host`（ホスト専用） | `init-host-env.sh` |
| `container`（コンテナ専用） | `sync-secrets.sh`, `validate-secrets.sh`, `sync-compose-secrets.sh` |
| `any`（どちらでも可） | 上記以外のすべて |

```
┌───────────────────────────────────────────────────┐
│ .sandbox/scripts/                                 │
│  ├── validate-secrets.sh  ← 組み込み（container） │
│  ├── test-*.sh            ← テストカテゴリ        │
│  ├── _startup_common.sh   ← スキップ（ライブラリ）│
│  └── my-script.sh         ← ファイルを置くだけ    │
│                                                   │
│ SandboxMCP が *.sh ファイルを自動検出             │
│ 登録や設定は不要                                  │
└───────────────────────────────────────────────────┘
```

AI アシスタントは `list_scripts` でスクリプトを発見し、`get_script_info` で使い方を確認し、`run_script` で実行できます。
