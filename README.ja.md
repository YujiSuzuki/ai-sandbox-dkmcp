# AI Sandbox Environment + DockMCP

[English README is here](README.md)


AIコーディングアシスタントのための、セキュアな開発環境テンプレートです。

- **秘匿情報を隠してAIを使う** — `.env` や秘密鍵はAIから見えないが、アプリは普通に動く
- **複数プロジェクトをまとめて扱う** — モバイル・API・Webを1つの環境でAIに見せる
- **他のコンテナにもアクセスできる** — DockMCPを使えば、AIが別コンテナのログ確認やテスト実行もできる

必要なものは **Docker** と **VS Code** だけ。[CLIだけでも使えます](docs/reference.ja.md#2つの環境)。

本プロジェクトはローカル開発環境での使用を想定しており、本番環境での使用は想定されていません。制約事項については「[この環境が解決していない課題](#この環境が解決していない課題)」と「[よくある質問](#よくある質問)」を参照してください。

> [!NOTE]
> **DockMCP単体での利用は非推奨です。** ホストOSでAIを実行する場合、AIはユーザーと同じ権限を持つため DockMCP を経由するメリットがありません。スタンドアロンセットアップについては [dkmcp/README.ja.md](dkmcp/README.ja.md) を参照してください。


---

# 目次

- [この環境が解決する実際の課題](#この環境が解決する実際の課題)
- [ユースケース](#ユースケース)
- [クイックスタート](#クイックスタート)
- [コマンド](#コマンド)
- [プロジェクト構造](#プロジェクト構造)
- [セキュリティ機能](#セキュリティ機能)
- [対応AIツール](#対応aiツール)
- [よくある質問](#よくある質問)
- [ドキュメント](#ドキュメント)



# この環境が解決する実際の課題

**秘匿情報の保護** — ホストOSでAIを実行すると `.env` や秘密鍵へのアクセスを防ぐのが困難です。本環境ではAIをDockerコンテナに隔離し、**コードは見えるけど秘匿ファイルは見えない** 状態を作ります。

**複数プロジェクトの横断開発** — アプリとサーバーの連携部分の不具合調査は大変です。本環境は複数プロジェクトを1つのワークスペースにまとめ、AIがシステム全体を見渡せるようにします。

**コンテナ間の連携** — Sandbox化すると他のコンテナにアクセスできなくなりますが、DockMCPがこれを解消します。AIがAPIコンテナのログを読んだり、テストを実行したりできます。

## この環境が解決していない課題

**ネットワーク制限** — AIの外部アクセスを制限したい場合は、プロキシの導入を検討してください。参考: [Docker Compose ネットワーク設定](https://docs.docker.com/compose/networking/) / [Anthropic公式のサンドボックス環境](https://github.com/anthropics/claude-code/tree/main/.devcontainer)


# ユースケース

### マイクロサービス開発
```
workspace/
├── mobile-app/     ← Flutter/React Native
├── api-gateway/    ← Node.js
├── auth-service/   ← Go
└── db-admin/       ← Python
```
APIキーを公開せずに、AIがすべてのサービスを横断してサポート。

### フルスタックプロジェクト
```
workspace/
├── frontend/       ← React
├── backend/        ← Django
└── workers/        ← Celeryタスク
```
AIがフロントエンドのコードを編集しながら、バックエンドのログを確認可能。

### レガシー + 新規
```
workspace/
├── legacy-php/     ← 古いコードベース
└── new-service/    ← モダンな書き直し
```
AIが両方を理解し、移行を支援。

---

# クイックスタート

## 前提条件

| 構成 | 必要なもの |
|------|-----------|
| **Sandbox（VS Code）** | Docker + VS Code + [Dev Containers拡張](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) |
| **Sandbox（CLIのみ）** | Docker のみ |
| **Sandbox + DockMCP** | 上記いずれか + [DockMCP](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases)（またはソースビルド）+ MCP対応AI CLI |

## しくみ（概要）

```
AI Sandbox（コンテナ）  →  DockMCP（ホストOS）  →  他のコンテナ（API, DB等）
   AIはここで動く            アクセスを中継           ログ確認・テスト実行
   秘匿ファイルは見えない     セキュリティポリシー適用
```

AIはDockerコンテナのSandbox内で動くため、秘匿ファイルはまるで存在しないかのようにアクセスできなくなります。それでも開発に支障はありません。DockMCPを通じて、AIは他のコンテナのログ確認やテスト実行ができるからです。

→ 詳しい構成図は [アーキテクチャ詳細](docs/architecture.ja.md) を参照

> **💡 日本語環境にする場合:** DevContainer（または cli_sandbox）を開く前に、ホストOS上で以下を実行：
> ```bash
> .sandbox/scripts/init-host-env.sh -i
> ```
> 言語選択で `2) 日本語` を選ぶと、コンテナ内のターミナル出力が日本語になります。
> （コンテナ内からでも実行できます）


## オプションA: Sandbox

秘匿情報の隠蔽だけでよい場合（DockMCPなし）：

```bash
# 1. VS Codeで開く
code .

# 2. コンテナで再度開く（Cmd+Shift+P / F1 → "Dev Containers: Reopen in Container"）
```

<details>
<summary><code>code</code> コマンドが見つからない場合</summary>

**VS Codeのメニューから開く方法:**
「ファイル → フォルダーを開く」でこのフォルダーを選択してください。

**`code` コマンドをインストールする方法（macOS）:**
VS Code上でコマンドパレット（Cmd+Shift+P）を開き、`Shell Command: Install 'code' command in PATH` を実行してください。ターミナルを再起動すると `code` コマンドが使えるようになります。

> 参考: [Visual Studio Code on macOS - 公式ドキュメント](https://code.visualstudio.com/docs/setup/mac)

</details>

<details>
<summary>CLI Sandbox 環境（ターミナルベース）の場合</summary>

```bash
   ./cli_sandbox/claude.sh # (Claude Code)
   ./cli_sandbox/gemini.sh # (Gemini CLI)
```

</details>

**これだけです！** AIは `/workspace` 内のコードにアクセスできますが、`.env` と `secrets/` ディレクトリは隠されています。


## オプションB: Sandbox + DockMCP

AIに他コンテナのログ確認やテスト実行もさせたい場合：

### ステップ1: DockMCPサーバーを起動（ホストOS上で）

```bash
cd dkmcp
make install        # ~/go/bin/ にインストール
dkmcp serve --config configs/dkmcp.example.yaml
```

> Go環境の構築は [Go公式サイト](https://go.dev/dl/) を参照。`make build` ではなく `make install` を使用してください。

<details>
<summary>ホストOSにGo環境がない場合</summary>

AI Sandbox内にはGo環境があるため、ホストOS向けのバイナリをクロスビルドできます。

```bash
# AI Sandbox内でビルド
cd /workspace/dkmcp
make build-host

# ホストOS上でインストール
cd <このリポジトリのパス>/dkmcp
make install-host DEST=~/go/bin        # Go環境がある場合
make install-host DEST=/usr/local/bin  # Go環境がない場合
```

</details>

### ステップ2: DevContainerを開く

```bash
code .
# Cmd+Shift+P / F1 → "Dev Containers: Reopen in Container"
```

### ステップ3: Claude CodeをDockMCPに接続

AI Sandbox内のシェルで：

```bash
claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse
```

Claude Codeで `/mcp` → 「Reconnect」を実行してください。

> **重要:** DockMCPサーバーを再起動した場合も、`/mcp` → 「Reconnect」が必要です。

### ステップ4（推奨）: カスタムドメイン設定

```bash
# macOS/Linux — ホストOS上で実行
echo "127.0.0.1 securenote.test api.securenote.test" | sudo tee -a /etc/hosts
```

> AI Sandboxは `docker-compose.yml` の `extra_hosts` により、カスタムドメインを自動的に解決します。

### ステップ5（オプション）: デモアプリで試す

```bash
# ホストOS上で
cd demo-apps
docker-compose -f docker-compose.demo.yml up -d --build
```

**アクセス:**
- Web: http://securenote.test:8000
- API: http://api.securenote.test:8000/api/health

**AIに話しかけてみる:**
- `securenote-apiのログを見せて`
- `securenote-apiでnpm testを実行して`
- `秘匿ファイルがあるか確認してみて`

→ 接続できない場合は [トラブルシューティング](docs/reference.ja.md#トラブルシューティング) を参照

## 次のステップ

- **セキュリティ機能を体験したい** → [ハンズオン](docs/hands-on.ja.md)
- **自分のプロジェクトで使いたい** → [自分のプロジェクトへの適用](docs/customization.ja.md)

---


# コマンド

| コマンド | 実行場所 | 説明 |
|---------|---------|------|
| `dkmcp serve` | ホストOS | DockMCPサーバーを起動 |
| `dkmcp list` | ホストOS | アクセス可能なコンテナを一覧表示 |
| `dkmcp client list` | AI Sandbox | HTTP経由でコンテナ一覧 |
| `dkmcp client logs <container>` | AI Sandbox | HTTP経由でログ取得 |
| `dkmcp client exec <container> "cmd"` | AI Sandbox | HTTP経由でコマンド実行 |

> 詳細なコマンドオプションについては [dkmcp/README.ja.md](dkmcp/README.ja.md#cliコマンド) を参照



# プロジェクト構造

`.sandbox/` に共有基盤、`.devcontainer/` と `cli_sandbox/` に2つのSandbox環境、`dkmcp/` にMCPサーバー、`demo-apps/` と `demo-apps-ios/` にデモアプリが配置されています。

<details>
<summary>ディレクトリツリーを見る</summary>

```
workspace/
├── .sandbox/               # 共有サンドボックス基盤
│   ├── Dockerfile          # コンテナイメージ定義
│   └── scripts/            # 共有スクリプト（validate-secrets, check-secret-sync, sync-secrets）
│
├── .devcontainer/          # VS Code Dev Container 設定
│   ├── docker-compose.yml  # 秘匿情報隠蔽の設定
│   └── devcontainer.json   # VS Code統合設定（拡張機能、ポート制御等）
│
├── cli_sandbox/             # CLI サンドボックス（代替環境）
│   ├── claude.sh           # ターミナルから Claude Code を実行
│   ├── gemini.sh           # ターミナルから Gemini CLI を実行
│   ├── ai_sandbox.sh       # 汎用シェル（AI なしでデバッグ用）
│   └── docker-compose.yml
│
├── dkmcp/               # コンテナアクセス用MCPサーバー
│   ├── cmd/dkmcp/
│   ├── internal/
│   ├── configs/
│   └── README.md
│
├── demo-apps/              # サーバーサイドプロジェクト
│   ├── securenote-api/     # Node.js バックエンド
│   ├── securenote-web/     # React フロントエンド
│   └── docker-compose.demo.yml
│
└── demo-apps-ios/          # iOSアプリプロジェクト
    ├── SecureNote/         # SwiftUI ソースコード
    ├── SecureNote.xcodeproj
    └── README.md
```

</details>

実際に使用する時は、デモアプリ demo-apps/ と demo-apps-ios/ を削除して使用します。詳しくは [自分のプロジェクトへの適用](docs/customization.ja.md) を参照。


# セキュリティ機能

| 機能 | やっていること |
|------|--------------|
| **秘匿情報の隠蔽** | `.env` や `secrets/` をDockerマウントでAIから隠す。アプリ側は普通に読める |
| **コンテナアクセス制御** | DockMCPがセキュリティポリシーに基づき、AIのアクセス範囲を制限 |
| **Sandbox保護** | 非rootユーザー、制限されたsudo、ホストOSのファイルにアクセス不可 |
| **出力マスキング** | ログに含まれるパスワードやAPIキーをDockMCPが自動マスク |

→ 各機能の詳細・設定方法は [アーキテクチャ詳細](docs/architecture.ja.md) を参照

> [!NOTE]
> **デモ環境での git status について:** このテンプレートではデモ用の秘匿ファイルを `git add -f` で強制追跡しているため、AI Sandbox 内の git status で「削除された」ように見えます。自分のプロジェクトでは秘匿ファイルを `.gitignore` に入れるため、この問題は発生しません。対処方法は [ハンズオン](docs/hands-on.ja.md) を参照してください。


# 対応AIツール

- ✅ **Claude Code** (Anthropic) - 完全なMCPサポート
- ✅ **Gemini Code Assist** (Google) - Agentモードで MCP対応（`.gemini/settings.json` にMCPの設定を記述する）
- ✅ **Gemini CLI** (Google) - ターミナル （MCP連携やIDE連携の対応状況は不明なため公式サイトを参照してください）
- ✅ **Cline** (VS Code拡張) - MCP統合（おそらく対応しています。未検証）



# よくある質問

**Q: なぜAIに `docker-compose up/down` を頼めないの？**
A: これは意図的な設計です。AIは「観察と提案」、人間は「インフラ操作の実行」という責任分離をしています。詳細は [DockMCP設計思想](dkmcp/README.ja.md#設計思想) を参照してください。

**Q: DockMCPを使う必要がありますか？**
A: いいえ。DockMCPなしでも通常のサンドボックスとして機能します。DockMCPはクロスコンテナアクセスを可能にします。

**Q: 本番環境で使用しても安全ですか？**
A: **いいえ、推奨しません。** DockMCPには認証機能がないため、ローカル開発環境での使用を想定しています。

**Q: 別の秘匿情報管理を使えますか？**
A: はい！他の秘匿情報管理の方法とも組み合わせられます。

**Q: Windowsでも動作しますか？**
A: Docker Desktop があれば動作する想定ですが、macOS でのみ確認済みです。Linux/Windows は未検証です。



# ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [ハンズオン](docs/hands-on.ja.md) | セキュリティ機能を実際に体験する演習 |
| [自分のプロジェクトへの適用](docs/customization.ja.md) | テンプレートのカスタマイズ手順 |
| [リファレンス](docs/reference.ja.md) | 環境設定、オプション、トラブルシューティング |
| [アーキテクチャ詳細](docs/architecture.ja.md) | セキュリティの仕組みと構成図 |
| [DockMCP ドキュメント](dkmcp/README.ja.md) | MCPサーバーの詳細 |
| [DockMCP 設計思想](dkmcp/README.ja.md#設計思想) | なぜDockMCPはコンテナ操作をサポートしないのか |
| [プラグインガイド](docs/plugins.ja.md) | マルチリポ構成でのClaude Codeプラグイン活用 |
| [デモアプリガイド](demo-apps/README.ja.md) | SecureNoteデモの実行方法 |
| [CLI Sandbox ガイド](cli_sandbox/README.ja.md) | ターミナルベースのサンドボックス |

## ライセンス

MIT License - [LICENSE](LICENSE) を参照
