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
│  │    └─ Claude Code / Gemini                   │ │   │
│  │       secrets/ → empty (hidden)              │ │   │
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
