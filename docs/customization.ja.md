# 自分のプロジェクトへの適用

このプロジェクトを始める方法は3つあります。状況に合ったものを選んでください。

[← README に戻る](../README.ja.md)

---

## プロジェクトの入手方法

| 方法 | GitHub アカウント | 更新方法 |
|---|---|---|
| **Use this template** | 必要 | [更新ガイド](updating.ja.md) を参照 |
| **git clone** | 不要 | `git pull origin main` |
| **ZIP ダウンロード** | 不要 | 新しい ZIP をダウンロードして手動適用 |

### 方法1: GitHub テンプレートとして使う（推奨）

GitHub で **「Use this template」** → **「Create a new repository」** をクリック。

作成されるリポジトリの特徴：
- テンプレートの全ファイル（このリポジトリのコミット履歴なし）
- 新しい Git 履歴からスタート
- アップストリームとは独立（自動同期なし）

作成後、新しいリポジトリをクローン：

```bash
git clone https://github.com/your-username/your-new-repo.git
cd your-new-repo
```

テンプレートから作成したリポジトリはアップストリームと自動同期されないため、**更新通知機能**を搭載しています。AI Sandbox 起動時に新しいリリースをチェックします。更新の適用方法は[更新ガイド](updating.ja.md)を参照してください。

### 方法2: 直接クローン

Git で上流の変更を直接追跡したい場合（コントリビュート目的など）：

```bash
git clone https://github.com/YujiSuzuki/ai-sandbox-dkmcp.git
cd ai-sandbox-dkmcp
```

`git pull origin main` で更新を取得できます。

### 方法3: ZIP ダウンロード

Git を使わない場合は、GitHub から ZIP をダウンロード（**「Code」** → **「Download ZIP」**）して展開します。

この方法では、更新時に新しい ZIP をダウンロードして変更を手動で適用する必要があります。新バージョンがリリースされると、起動時の更新通知で知ることができます。

---

## プロジェクトのカスタマイズ

テンプレートを使用した場合も直接クローンした場合も、以下の手順で環境をカスタマイズします。

### AI アシスタントによるセットアップ

AI Sandbox 環境で AI アシスタントにカスタマイズを任せることができます。AI Sandbox を開いて、プロジェクトの情報を伝えてください：

> 「このテンプレートを自分のプロジェクトに合わせてカスタマイズして。構成は：
> - `my-api/`（Node.js API、`.env` と `secrets/` ディレクトリあり）
> - `my-web/`（React フロントエンド、秘匿ファイルなし）
>
> コンテナ名は `my-api`、`my-web`。my-api で許可するコマンド: `npm test`, `npm run lint`」

AI が docker-compose.yml の編集、dkmcp.yaml の作成、AI 設定ファイルの更新、バリデーションスクリプトの実行を行います。DevContainer のリビルドと DockMCP の起動だけはユーザー自身で行ってください。

以下は手動でセットアップする場合の手順です。

### demo-apps を自分のプロジェクトに置き換え

```bash
# デモアプリを削除（または参考用に残す）
rm -rf demo-apps demo-apps-ios

# 自分のプロジェクトを配置
git clone https://github.com/your-org/your-api.git
git clone https://github.com/your-org/your-web.git
```

### 秘匿ファイルの隠蔽設定

**`.devcontainer/docker-compose.yml`** と **`cli_sandbox/docker-compose.yml`** の両方を編集：

```yaml
services:
  ai-sandbox:
    volumes:
      # 秘匿ファイルを隠す（/dev/null にマウント）
      - /dev/null:/workspace/your-api/.env:ro
      - /dev/null:/workspace/your-api/config/secrets.json:ro

    tmpfs:
      # 秘匿ディレクトリを空にする
      - /workspace/your-api/secrets:ro
      - /workspace/your-api/keys:ro
```

**ポイント:**
- `.env` ファイル → `/dev/null` にマウント
- `secrets/` ディレクトリ → `tmpfs` + `:ro` で空のディレクトリに
- 両方の docker-compose.yml を同じ設定にする

**自動検証:**

起動時に以下のチェックが自動実行されます：
1. `validate-secrets.sh` - 秘匿が実際に機能しているか検証（docker-compose.yml からパスを自動読み込み）
2. `compare-secret-config.sh` - DevContainer と CLI の設定に差異があれば警告
3. `check-secret-sync.sh` - AI設定でブロックされたファイルが docker-compose.yml で隠蔽されていなければ警告
   - 対応: `.claude/settings.json`, `.aiexclude`, `.geminiignore`
   - 注意: `.gitignore` は意図的に**非対応**です — 秘匿情報以外のパターン（`node_modules/`, `dist/`, `*.log`）が多く含まれノイズになります。AI専用ファイルに秘匿情報のみを明示的に記載してください。

**手動同期ツール:** `check-secret-sync.sh` で未設定ファイルが報告された場合、`.sandbox/scripts/sync-secrets.sh` を実行して対話的に追加できます。オプション `4`（プレビュー）でファイルを変更せずに設定内容を確認できます。

**初回セットアップの推奨フロー:**
```bash
# 1. AIなしでコンテナに入る（AIは自動起動しない）
./cli_sandbox/ai_sandbox.sh

# 2. コンテナ内で: 対話的に秘匿設定を同期
.sandbox/scripts/sync-secrets.sh

# 3. 終了して DevContainer をリビルド
exit
# その後 VS Code で DevContainer を開く
```

これにより、AIがファイルにアクセスする前に秘匿設定が完了します。

判定ルール：
- `/dev/null:/workspace/...` の volumes → 秘匿ファイル
- `/workspace/...:ro` の tmpfs → 秘匿ディレクトリ

### DockMCP設定

**`dkmcp/configs/dkmcp.example.yaml`** をコピーして編集：

```bash
cp dkmcp/configs/dkmcp.example.yaml dkmcp.yaml
```

```yaml
security:
  mode: "moderate"

  # あなたのコンテナ名に変更
  allowed_containers:
    - "your-api-*"
    - "your-web-*"
    - "your-db-*"

  # 許可するコマンドを設定
  exec_whitelist:
    "your-api":
      - "npm test"
      - "npm run lint"
      - "python manage.py test"
    "your-db":
      - "psql -c 'SELECT 1'"
```

より厳格な設定例：

```yaml
security:
  mode: "strict"  # 読み取り専用（logs, inspect, stats）

  allowed_containers:
    - "prod-*"      # 本番コンテナのみ

  exec_whitelist: {}  # コマンド実行なし
```

複数インスタンスの起動など、詳細は [dkmcp/README.ja.md「サーバー起動」](../dkmcp/README.ja.md#複数インスタンスの起動) を参照。

### AI アシスタントの設定

AI アシスタントがプロジェクトの構成や秘匿ポリシーを正しく理解できるよう、以下のファイルを編集します。

**自動で反映されるもの（手順不要）:**

サブプロジェクトに `.claude/settings.json` が既にある場合、AI Sandbox 起動時に自動マージされます（`merge-claude-settings.sh`）。新規に作成する必要はありません。

**編集が必要なもの:**

| ファイル | 内容 | 対応 |
|----------|------|------|
| `CLAUDE.md` | Claude Code 向けのプロジェクト説明 | デモアプリ固有の記述を削除し、自分のプロジェクトに書き換え |
| `GEMINI.md` | Gemini Code Assist 向けのプロジェクト説明 | 同上 |
| `.aiexclude` | Gemini Code Assist の秘匿パターン | 必要に応じて自分の秘匿パスを追加 |
| `.geminiignore` | Gemini CLI の秘匿パターン | 同上 |

**CLAUDE.md / GEMINI.md の編集方針:**

- **残す**: DockMCP MCP Tools の使い方、セキュリティアーキテクチャの概要、環境の分離（What Runs Where）
- **書き換え**: プロジェクト構造、Common Tasks の具体例
- **削除**: SecureNote デモ固有の記述、デモシナリオの説明

### プラグインの活用（マルチリポ構成）

マルチリポ構成（各プロジェクトが独立したGitリポジトリ）でClaude Codeプラグインを使う場合は工夫が必要です。詳細は [プラグインガイド](plugins.ja.md) を参照。

> **注意**: このセクションは Claude Code 専用です。Gemini Code Assist では使えません。

### DevContainerをリビルド

```bash
# VS Code で Command Palette を開く (Cmd/Ctrl + Shift + P)
# "Dev Containers: Rebuild Container" を実行
```

### 動作確認

```bash
# AI Sandbox内で秘匿ファイルが隠されていることを確認
cat your-api/.env
# → 空または "No such file"

# DockMCPでコンテナにアクセスできることを確認
# Claude Code に "コンテナ一覧を見せて" と聞く
# Claude Code に "your-apiのログを見せて" と聞く
```

### チェックリスト

- [ ] `.devcontainer/docker-compose.yml` で秘匿ファイルを設定
- [ ] `cli_sandbox/docker-compose.yml` で同じ設定を適用
- [ ] `dkmcp.yaml` でコンテナ名を設定
- [ ] `dkmcp.yaml` で許可コマンドを設定
- [ ] `CLAUDE.md` / `GEMINI.md` を自分のプロジェクトに合わせて編集
- [ ] `.aiexclude` / `.geminiignore` に秘匿パスを追加（必要に応じて）
- [ ] DevContainerをリビルド
- [ ] 秘匿ファイルが隠されていることを確認
- [ ] DockMCP経由でログ確認できることを確認
