# 自分のプロジェクトへの適用

このリポジトリは **GitHub テンプレートリポジトリ** として設計されています。テンプレートから自分のプロジェクトを作成できます。

[← README に戻る](../README.ja.md)

---

## テンプレートとして使う

### Step 1: テンプレートから作成

GitHub で **「Use this template」** → **「Create a new repository」** をクリック。

作成されるリポジトリの特徴：
- テンプレートの全ファイル（このリポジトリのコミット履歴なし）
- 新しいGit履歴からスタート
- アップストリームとは独立（自動同期なし）

### Step 2: 新しいリポジトリをクローン

```bash
git clone https://github.com/your-username/your-new-repo.git
cd your-new-repo
```

### 更新をチェック

テンプレートから作成したリポジトリはアップストリームの更新を自動で受け取れないため、**更新通知機能** を搭載しています。AI Sandbox 起動時に GitHub の新しいリリースをチェックし、新バージョンがあれば通知します。

<details>
<summary>通知の例と設定の詳細</summary>

**仕組み:**
- デフォルトでは **プレリリースを含む全リリース** をチェックするため、バグ修正や改善をすぐに受け取れます
- 初回起動時は最新バージョンを記録するだけで、通知は表示されません
- 2回目以降のチェックで新バージョンが見つかると、以下のような通知が表示されます：

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📦 更新チェック
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  現在のバージョン:  v0.1.0
  最新バージョン:   v0.2.0

  更新方法:
    1. リリースノートで変更内容を確認
    2. 必要な変更を手動で適用

  リリースノート:
    https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**更新の適用方法:**
1. [リリースノート](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases) で変更内容を確認
2. 必要な変更を手動でプロジェクトに適用

**設定ファイル:** `.sandbox/config/template-source.conf`
```bash
TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_CHANNEL="all"            # "all" = プレリリース含む, "stable" = 正式リリースのみ
CHECK_UPDATES="true"           # "false" で無効化
CHECK_INTERVAL_HOURS="24"      # チェック間隔（0 = 毎回）
```

| `CHECK_CHANNEL` | 動作 | ユースケース |
|---|---|---|
| `"all"`（デフォルト） | プレリリースを含む全リリースをチェック | バグ修正や改善をすぐに受け取りたい |
| `"stable"` | 正式リリースのみチェック | 安定版マイルストーンだけ追いたい |

</details>

---

## 別の方法: 直接クローン

Git で上流の変更を追跡したい場合（コントリビュート目的など）：

```bash
git clone https://github.com/YujiSuzuki/ai-sandbox-dkmcp.git
cd ai-sandbox-dkmcp
```

---

## プロジェクトのカスタマイズ

テンプレートを使用した場合も直接クローンした場合も、以下の手順で環境をカスタマイズします。

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
