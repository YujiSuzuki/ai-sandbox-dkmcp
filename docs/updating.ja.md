# テンプレートの更新

オリジナルのテンプレートから更新を取り込む方法を説明します。

[← README に戻る](../README.ja.md)

---

## 起動時の自動更新チェック

AI Sandbox の起動時に新しいリリースを自動チェックします。新バージョンがあると以下のような通知が表示されます：

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📦 更新チェック
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  現在のバージョン:  v1.1.0
  最新バージョン:   v1.2.0

  💡 AIに更新を依頼できます
   例: 「最新バージョンに更新して」

  リリースノート:
    https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 設定

更新チェックの設定は `.sandbox/config/template-source.conf` で管理します：

```bash
TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_UPDATES="true"           # "false" で無効化
CHECK_INTERVAL_HOURS="24"      # チェック間隔（0 = 毎回）
CHECK_CHANNEL="all"            # "all" = プレリリース含む, "stable" = 正式リリースのみ
```

| `CHECK_CHANNEL` | 動作 | ユースケース |
|---|---|---|
| `"all"`（デフォルト） | プレリリースを含む全リリースをチェック | バグ修正や改善をすぐに受け取りたい |
| `"stable"` | 正式リリースのみチェック | 安定版マイルストーンだけ追いたい |

---

## AI アシスタントで簡単に更新

もっとも手軽な方法は、AI アシスタントに頼むことです：

```
あなた: 「最新バージョンに更新して」
```

AI アシスタントが以下を行います：
1. 新バージョンの変更内容を確認
2. カスタマイズとの競合を検出
3. 変更内容と影響を説明
4. 確認を得てから更新を適用
5. 必要なコンポーネント（SandboxMCP等）をリビルド

Clone・テンプレートどちらの場合でも、AI アシスタントがセットアップ方法を判断して適切に対応します。

---

## 手動での更新

手順はプロジェクトのセットアップ方法によって異なります。

### リポジトリを直接 clone した場合

```bash
# 1. 新しい変更を確認
git fetch origin main
git log HEAD..origin/main --oneline

# 2. 変更を取得
git pull origin main

# 3. リビルド（下記「更新後の作業」参照）
```

### ZIP でダウンロードした場合

1. [リポジトリページ](https://github.com/YujiSuzuki/ai-sandbox-dkmcp)から最新の ZIP をダウンロード（**「Code」** → **「Download ZIP」**）
2. 新しいファイルと現在のプロジェクトを比較して、必要な変更を手動で適用
3. インフラ部分（`.sandbox/`、`dkmcp/`、`.devcontainer/`、`cli_sandbox/`）を重点的に確認

### GitHub テンプレートから作成した場合

テンプレートから作成したリポジトリには **upstream との自動的な接続がありません**。以下のいずれかの方法で更新を取り込みます。

#### 方法A: リリースノートを見て手動適用（最もシンプル）

1. [リリースノート](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases)で変更内容を確認
2. 必要な変更を手動でプロジェクトに適用

向いているケース：更新が小規模または頻度が低い場合、テンプレートから大きく変更している場合

#### 方法B: upstream remote を追加してマージ

元リポジトリをリモートとして追加し、変更を取り込みます：

```bash
# 初回のみ: テンプレートリポジトリを "upstream" として追加
git remote add upstream https://github.com/YujiSuzuki/ai-sandbox-dkmcp.git

# 更新を取得してマージ
git fetch upstream main
git merge upstream/main
```

> **注意:** テンプレートから大きくカスタマイズしている場合、マージ時にコンフリクトが発生する可能性があります。手動で解消してください。

向いているケース：テンプレートの構造にあまり手を加えていない場合

#### 方法C: GitHub Actions で自動同期

[actions-template-sync](https://github.com/AndreasAugustin/actions-template-sync) を使うと、テンプレートの更新を自動的にプルリクエストとして受け取れます：

```yaml
# .github/workflows/template-sync.yml
name: Template Sync
on:
  schedule:
    - cron: "0 0 * * 0"  # 毎週
  workflow_dispatch:

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: AndreasAugustin/actions-template-sync@v2
        with:
          source_repo_path: YujiSuzuki/ai-sandbox-dkmcp
          upstream_branch: main
```

テンプレートに変更があると PR が作成されるので、内容を確認してからマージできます。

向いているケース：最小限の手間で常に最新の状態を保ちたいチーム

---

## デモアプリの変更について

カスタマイズの際に `demo-apps/` や `demo-apps-ios/` を削除済みの場合（多くのユーザーがそうしているはずです）、更新に含まれるデモアプリの変更は無視して構いません。

- **方法A（手動）:** リリースノートのデモ関連の変更をスキップするだけ
- **方法B（upstream merge）:** 削除済みのデモファイルでコンフリクトが発生しますが、削除を維持する形で解消してください（`git rm` してマージを続行）
- **方法C（GitHub Actions）:** 生成される PR にデモアプリの変更が含まれますが、それらのファイルは PR から除外してください

更新で重要なのはインフラ部分（`.sandbox/`、`dkmcp/`、`.devcontainer/`、`cli_sandbox/`）です。

---

## 更新後の作業

どの方法で更新しても、変更内容に応じてコンポーネントのリビルドが必要です。

### SandboxMCP のリビルド（`.sandbox/sandbox-mcp/` が変更された場合）

```bash
# AI Sandbox 内で実行
cd .sandbox/sandbox-mcp
make clean && make register
```

### DockMCP のリビルド（`dkmcp/` が変更された場合）

```bash
# ホストOS上で実行（AI Sandbox内ではない）
cd dkmcp
make install
```

### VS Code の再起動または MCP の再接続

- **簡易**: Claude Code で `/mcp` → 「Reconnect」
- **完全**: VS Code を再起動（macOS: Cmd+Q / Windows/Linux: Alt+F4）

### リビルドが必要かどうかの判断

更新の差分を確認して、どのディレクトリが変更されたかを見ます：

```bash
# clone ユーザー（pull する前に）
git diff HEAD..origin/main --stat

# テンプレートユーザー（マージ後に）
git diff HEAD~1 --stat
```

| 変更されたディレクトリ | 必要な作業 |
|---|---|
| `.sandbox/sandbox-mcp/` | SandboxMCP をリビルド |
| `dkmcp/` | DockMCP をリビルド（ホストOS） |
| `.devcontainer/` | DevContainer をリビルド |
| `.sandbox/scripts/` | リビルド不要（直接使用される） |
