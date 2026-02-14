# CLI Sandbox

[English](README.md)

ターミナルからAIコーディングアシスタントを実行するための代替環境です。

基本的な使い方や、この環境が存在する理由（DevContainer が壊れた場合の復旧手段）については、[ルートの README.ja.md](../README.ja.md#2つの環境) を参照してください。

## ファイル構成

| ファイル | 役割 |
|---------|------|
| `claude.sh` | Claude Code をコンテナ内で起動 |
| `gemini.sh` | Gemini CLI をコンテナ内で起動 |
| `ai_sandbox.sh` | AI なしの対話シェルを起動（デバッグ・調査用） |
| `_common.sh` | 上記スクリプト共通の起動処理・バリデーション |
| `docker-compose.yml` | コンテナ定義（秘匿設定・リソース制限含む） |
| `.env.example` | 環境変数のテンプレート |
| `.env` | 環境変数の実設定（`.gitignore` 対象） |
| `build.sh` | イメージビルド |
| `build-no-cache.sh` | キャッシュなしイメージビルド |
| `test-sudo-security.sh` | sudo 制限が正しく効いているかの検証スクリプト |
| `.dockerignore` | ビルド時の除外対象 |

## 起動フロー

各スクリプト（`claude.sh`、`gemini.sh`、`ai_sandbox.sh`）は `_common.sh` を `source` して共通の処理を実行します。

```
スクリプト起動
  │
  ├─ 必須変数を設定（SCRIPT_NAME, COMPOSE_PROJECT_NAME, SANDBOX_ENV）
  ├─ _common.sh を source
  │    ├─ 必須変数の検証
  │    ├─ 実行ディレクトリの確認（cli_sandbox の親から実行されているか）
  │    └─ .env.sandbox, cli_sandbox/.env の読み込み
  │
  ├─ run_startup_scripts()
  │    ├─ merge-claude-settings.sh    … Claude 設定のマージ
  │    ├─ validate-secrets.sh        … シークレット隠蔽の検証
  │    ├─ compare-secret-config.sh    … DevContainer と CLI の設定差異チェック
  │    ├─ validate-secrets.sh         … 秘匿設定が機能しているか検証
  │    └─ check-secret-sync.sh        … .claude/settings.json との同期チェック
  │
  ├─ [検証成功] → AI ツール起動（claude / gemini / bash）
  └─ [検証失敗] → confirm_continue_after_failure()
       ├─ [y] シェルのみ起動（AI は起動しない）
       └─ [N] 終了
```

検証が失敗した場合、AI ツールは意図的に起動されません。シェルのみで入り、設定の修正を行ってからやり直す流れになります。

## 環境変数

### .env.example の設定項目

```bash
TERM=xterm-256color       # ターミナル種別
COLORTERM=truecolor       # カラー出力
SANDBOX_MEMORY_LIMIT=4gb  # コンテナのメモリ上限
```

注意： `COMPOSE_PROJECT_NAME` は各起動スクリプト内でデフォルト値が設定されています（`claude.sh` → `cli-claude`、`gemini.sh` → `cli-gemini` 等）。`.env` で設定すると上書きされ、全スクリプトで同じプロジェクト名が使われます

### SANDBOX_ENV

コンテナ内で現在の環境を識別するための変数です。スクリプトごとに異なる値が設定されます。

| スクリプト | SANDBOX_ENV の値 |
|-----------|-----------------|
| `claude.sh` | `cli_claude` |
| `gemini.sh` | `cli_gemini` |
| `ai_sandbox.sh` | `cli_ai_sandbox` |

## docker-compose.yml の構成

### 秘匿情報の隠蔽

DevContainer（`.devcontainer/docker-compose.yml`）と同じ秘匿設定を維持する必要があります。両者に差異があると起動時に `compare-secret-config.sh` が警告を出します。

```yaml
volumes:
  # ファイル単位の隠蔽: /dev/null にマウント → 空ファイルに見える
  - /dev/null:/workspace/demo-apps/securenote-api/.env:ro

tmpfs:
  # ディレクトリ単位の隠蔽: tmpfs で空ディレクトリに見える
  - /workspace/demo-apps/securenote-api/secrets:ro
```

秘匿設定の追加・同期については [ルートの README.ja.md「自分のプロジェクトへの適用」](../README.ja.md#自分のプロジェクトへの適用) を参照してください。

### リソース制限

コンテナがホストの資源を使い切らないよう制限を設けています。

```yaml
deploy:
  resources:
    limits:
      memory: ${SANDBOX_MEMORY_LIMIT:-4gb}
      cpus: "${SANDBOX_CPU_LIMIT:-2}"
```

`.env` で `SANDBOX_MEMORY_LIMIT` や `SANDBOX_CPU_LIMIT` を変更できます。

### ホームディレクトリの永続化

認証情報（`.claude.json`、`.claude/` 等）は名前付きボリューム `cli-sandbox-home` に保存されます。`COMPOSE_PROJECT_NAME` が異なると別のボリュームになるため、ツール間でホームディレクトリは共有されません。

ボリューム間のコピーが必要な場合は `.sandbox/host-tools/copy-credentials.sh` を使えます。詳細は [docs/reference.ja.md](../docs/reference.ja.md#ホームディレクトリのエクスポートインポート) を参照してください。

## セキュリティテスト

コンテナ内の sudo 制限が正しく機能しているかを確認するスクリプトです。

```bash
# コンテナに入る
./cli_sandbox/ai_sandbox.sh bash

# コンテナ内で実行
cd ./cli_sandbox
./test-sudo-security.sh
```

テスト内容:
- **許可されるべきコマンド**: `apt-get`、`apt`、`dpkg`、`pip3`、`npm`（パスワードなしで実行可能か）
- **拒否されるべきコマンド**: `rm`、`chmod`、`chown`、`su`、`bash`、`cat`、`mv`、`cp`（ブロックされるか）

ホストOS上で実行しようとするとエラーになります（コンテナ内のみ対応）。

## DevContainer との違い

| 項目 | DevContainer | CLI Sandbox |
|------|-------------|-------------|
| 起動方法 | VS Code から | ターミナルから `./cli_sandbox/*.sh` |
| IDE 連携 | VS Code 拡張機能あり | なし |
| Go 環境 | devcontainer.json の features で追加 | なし（必要なら手動インストール） |
| プロジェクト名 | `.devcontainer/.env` で設定 | スクリプトごとのデフォルト or `cli_sandbox/.env` |
| 用途 | 日常的な開発 | 復旧・代替・ターミナル作業 |
