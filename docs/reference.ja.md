# リファレンス

環境設定、オプション、トラブルシューティングなどの補足情報です。

[← README に戻る](../README.ja.md)

---

## 2つの環境

| 環境 | 用途 | 使用タイミング |
|-------------|---------|-------------|
| **DevContainer** (`.devcontainer/`) | VS Codeでの主要開発 | 日常的な開発 |
| **CLI Sandbox** (`cli_sandbox/`) | 代替/復旧 | DevContainerが壊れた時 |

**なぜ2つの環境？**

**復旧用の代替環境** として重要です。

Dev Container の設定が壊れた場合：
1. VS Code で Dev Container が起動できない
2. Claude Code も動かない
3. 設定を直すのに AI の助けを借りられない → **詰む**

`cli_sandbox/` があれば：
1. Dev Container が壊れても
2. ホストから AI を起動できる
   - `./cli_sandbox/claude.sh` (Claude Code)
   - `./cli_sandbox/gemini.sh` (Gemini CLI)
3. AI に Dev Container の設定を直してもらえる

```bash
./cli_sandbox/claude.sh   # または
./cli_sandbox/gemini.sh
# 壊れたDevContainer設定をAIに修正してもらう
```

---

## プロジェクト名のカスタマイズ

デフォルトでは、DevContainer のプロジェクト名は `<親ディレクトリ名>_devcontainer`（例：`workspace_devcontainer`）になります。

カスタムのプロジェクト名を設定するには、`.devcontainer/.env` ファイルを作成します：

```bash
# .env.example をコピー
cp .devcontainer/.env.example .devcontainer/.env
```

`.env` ファイルの内容：
```bash
COMPOSE_PROJECT_NAME=ai-sandbox
```

これにより、コンテナ名やボリューム名がより分かりやすくなります：
- コンテナ: `ai-sandbox-ai-sandbox-1`
- ボリューム: `ai-sandbox_node-home`

> **注意:** `.env` ファイルは `.gitignore` に追加されているため、各開発者が自分用の設定を持てます。

---

## 起動時の自動検証

DevContainer と CLI Sandbox は、起動するたびに以下の検証を自動実行します：

| チェック内容 | やっていること |
|-------------|--------------|
| AI設定のマージ | サブプロジェクトの `.claude/settings.json` を自動統合 |
| 設定の整合性 | DevContainer と CLI Sandbox で秘匿設定にズレがないか確認 |
| 秘匿ファイルの隠蔽 | `.env` や `secrets/` が実際にAIから見えなくなっているか検証 |
| 同期チェック | AI設定でブロックしたファイルが docker-compose でも隠されているか確認 |
| テンプレート更新 | 新しいバージョンのテンプレートがあれば通知 |

問題が見つかった場合は警告が表示され、ユーザーが確認してから続行できます。<ins>設定ミスに気づかないまま作業してしまう心配がありません。</ins>

### 出力オプション

検証結果の表示量を制御できます：

| モード | フラグ | 出力内容 |
|--------|--------|----------|
| Quiet | `--quiet` または `-q` | 警告とエラーのみ（最小限） |
| Summary | `--summary` または `-s` | 簡潔なサマリー |
| Verbose | (なし、デフォルト) | 罫線装飾付きの詳細出力 |

**CLI Sandbox の例：**
```bash
# 最小限の出力（警告のみ）
./cli_sandbox/ai_sandbox.sh --quiet

# 簡潔なサマリー
./cli_sandbox/ai_sandbox.sh --summary
```

**環境変数：**
```bash
# デフォルトの詳細度を設定
export STARTUP_VERBOSITY=quiet  # または: summary, verbose
```

**設定ファイル:** `.sandbox/config/startup.conf`
```bash
# 全起動スクリプトのデフォルト詳細度
STARTUP_VERBOSITY="verbose"

# "詳細はREADMEを参照"メッセージで使用するURL
README_URL="README.md"
README_URL_JA="README.ja.md"  # LANG=ja_JP* の場合に使用

# ラベルごとのバックアップ保持件数（0 = 無制限）
BACKUP_KEEP_COUNT=0
```

sync スクリプトが作成するバックアップは `.sandbox/backups/` に保存されます。保持件数を制限するには：

```bash
# 直近10件のみ保持
BACKUP_KEEP_COUNT=10

# 環境変数で一時的に上書きも可能
BACKUP_KEEP_COUNT=10 .sandbox/scripts/sync-secrets.sh
```

---

## 同期警告からのファイル除外

起動スクリプトは `.claude/settings.json` でブロックされたファイルが `docker-compose.yml` でも隠蔽されているかチェックします。特定のパターン（`.example` ファイルなど）を警告から除外するには、`.sandbox/config/sync-ignore` を編集します：

```gitignore
# example/template ファイルを同期警告から除外
**/*.example
**/*.sample
**/*.template
```

これは gitignore 形式のパターンを使用します。これらのパターンにマッチするファイルは「docker-compose.yml に未設定」警告をトリガーしません。

---

## 複数のDevContainerを起動する場合

完全に分離したDevContainer環境が必要な場合（例：異なるクライアント案件）、`COMPOSE_PROJECT_NAME` を使って分離したインスタンスを作成できます。

<details>
<summary>方法とホームディレクトリの共有</summary>

### 方法A: .env ファイルで分離（推奨）

`.devcontainer/.env` で異なるプロジェクト名を設定：

```bash
COMPOSE_PROJECT_NAME=client-a
```

別のワークスペースでは：

```bash
COMPOSE_PROJECT_NAME=client-b
```

### 方法B: コマンドラインで分離

異なるプロジェクト名でDevContainerを起動：

```bash
# プロジェクトA
COMPOSE_PROJECT_NAME=client-a docker-compose up -d

# プロジェクトB（別のボリュームが作成される）
COMPOSE_PROJECT_NAME=client-b docker-compose up -d
```

> ⚠️ **注意:** プロジェクト名が異なるとボリュームも別になるため、ホームディレクトリ（認証情報・設定・履歴）は自動的に共有されません。下記「ホームディレクトリのコピー」を参照。

### 方法C: バインドマウントでホームディレクトリを共有

全インスタンスでホームディレクトリを自動共有したい場合、`docker-compose.yml` をバインドマウントに変更：

```yaml
volumes:
  # 名前付きボリュームの代わりにバインドマウント
  - ~/.ai-sandbox/home:/home/node
  - ~/.ai-sandbox/gcloud:/home/node/.config/gcloud
```

**メリット:**
- 全インスタンスでホームディレクトリを自動共有
- バックアップが簡単（ホストディレクトリをコピーするだけ）

**デメリット:**
- ホストのディレクトリ構造に依存
- Linuxホストでは UID/GID の調整が必要な場合あり

### ホームディレクトリのエクスポート/インポート

ホームディレクトリ（認証情報・設定・履歴）をバックアップまたは別のワークスペースに移行できます：

```bash
# ワークスペース全体をエクスポート（devcontainer と cli_sandbox の両方）
./.sandbox/host-tools/copy-credentials.sh --export /path/to/workspace ~/backup

# 特定の docker-compose.yml からエクスポート
./.sandbox/host-tools/copy-credentials.sh --export .devcontainer/docker-compose.yml ~/backup

# ワークスペースにインポート
./.sandbox/host-tools/copy-credentials.sh --import ~/backup /path/to/workspace
```

**注意:** インポート先のボリュームが存在しない場合、先に環境を一度起動してボリュームを作成する必要があります。

用途：
- `~/.claude/` の使用量データを確認
- 設定のバックアップ
- 新しいワークスペースへの認証情報の移行
- トラブルシューティング

</details>

---

## DockMCPのアンインストール

DockMCPが不要になった場合、インストール先に応じてバイナリを削除します：

```bash
rm ~/go/bin/dkmcp
# または
rm /usr/local/bin/dkmcp
```

---

## トラブルシューティング

### DockMCP接続

Claude CodeがDockMCPツールを認識しない場合：

1. **VS Codeのポートパネルを確認** - DockMCPのポート（デフォルトでは8080）がフォワードされていたら停止
2. **DockMCPが実行中か確認** - `curl http://localhost:8080/health`（ホストOS上で）
3. **MCP再接続を試す** - Claude Codeで `/mcp` を実行し、「Reconnect」を選択
4. **VS Codeを完全に再起動**（Cmd+Q / Alt+F4）- Reconnectで解決しない場合

### フォールバック：AI Sandbox内でdkmcp clientを使用

MCPプロトコルが動作しない場合（Claude CodeやGeminiが接続できない）、フォールバックとしてAI Sandbox内で `dkmcp client` コマンドを直接使用できます。

> **注意:** `/mcp` で「✔ connected」と表示されていても、MCPツールが「Client not initialized」エラーで失敗することがあります。これはVS Code拡張機能（Claude Code, Gemini Code Assist等）のセッション管理のタイミング問題が原因である可能性があります。この場合：
> 1. まず `/mcp` → 「Reconnect」を試す（簡単な解決策）
> 2. それでも解決しない場合、AIは `dkmcp client` コマンドをフォールバックとして使用
> 3. 最終手段として、VS Codeを完全に再起動して接続を再確立

**セットアップ（初回のみ）:**

AI Sandbox内でdkmcpをインストール：
```bash
cd /workspace/dkmcp
make install
```

> **注意:** Go環境はデフォルトで有効です。インストール後、イメージサイズを小さくしたい場合は `.devcontainer/devcontainer.json` の `features` ブロックをコメントアウトしてリビルドできます。

**使用方法:**
```bash
# コンテナ一覧
dkmcp client list

# ログ取得
dkmcp client logs securenote-api

# コマンド実行
dkmcp client exec securenote-api "npm test"
```

> **`--url` について:** デフォルトで `http://host.docker.internal:8080` に接続します。`dkmcp.yaml` でサーバーのポートを変更した場合は、`--url` フラグまたは環境変数 `DOCKMCP_SERVER_URL` で明示的に指定してください。
> ```bash
> dkmcp client list --url http://host.docker.internal:9090
> # または
> export DOCKMCP_SERVER_URL=http://host.docker.internal:9090
> ```
