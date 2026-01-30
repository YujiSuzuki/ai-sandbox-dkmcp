# AI Sandbox Environment + DockMCP

[English README is here](README.md)


本プロジェクトはセキュリティリスクを最小限にとどめ、AIの持つ俯瞰的な分析能力をフルに引き出すことを目指した開発環境テンプレートです。

必要なものは Docker（または OrbStack）と VS Code です。（ただし VS Code は必須ではありません。[CLIだけでも使えます](#2つの環境)）


- **複数プロジェクトの横断開発** — モバイル・API・Webなど複数のコードベースを1つの環境でAIに扱わせる
- **秘匿情報の構造的な隔離** — `.env` や秘密鍵をボリュームマウントでAIから隠しつつ、実コンテナでは通常どおり利用
- **DockMCPによるコンテナ間連携** — AI が別コンテナのログ確認やテスト実行を行える


本プロジェクトはローカル開発環境での使用を想定しており、本番環境での使用は想定されていません。制約事項については「[この環境が解決していない課題](#この環境が解決していない課題)」と「[よくある質問](#よくある質問)」を参照してください。



> [!NOTE]
> **DockMCP単体での利用は非推奨です。** ホストOSでAIを実行する場合、AIはユーザーと同じ権限を持つため DockMCP を経由するメリットがありません。リモートホストへのアクセス用途も考えられますが、現時点では認証機能がないためアクセス制限がありません。スタンドアロンセットアップについては [dkmcp/README.ja.md](dkmcp/README.ja.md) を参照してください。


---

# 目次

- [この環境が解決する実際の課題](#この環境が解決する実際の課題)
- [ユースケース](#ユースケース)
- [クイックスタート](#クイックスタート)
- [コマンド](#コマンド)
- [プロジェクト構造](#プロジェクト構造)
- [セキュリティ機能](#セキュリティ機能)
- [デモを試す](#デモを試す)
- [2つの環境](#2つの環境)
- [高度な使い方](#高度な使い方)
- [自分のプロジェクトへの適用](#自分のプロジェクトへの適用)
  - [テンプレートとして使う](#テンプレートとして使う)
    - [更新をチェック](#更新をチェック)
  - [または、直接クローン](#別の方法-直接クローン)
- [対応AIツール](#対応aiツール)
- [よくある質問](#よくある質問)
- [ドキュメント](#ドキュメント)






# この環境が解決する実際の課題

## 秘匿情報の構造的な保護
ホストOSでのAI実行は便利ですが、.env や秘密鍵へのアクセスを防ぐのは困難です。本環境ではAIをSandboxコンテナ内に隔離し、ボリュームマウント制御によって**「コードは見えても、秘密は見えない」**境界線を構造的に構築します。

## 複数プロジェクトの横断的な統合
リポジトリ間の「隙間」で発生する不具合の調査は、人間のエンジニアにとっても重労働です。本環境は複数のプロジェクトを1つのワークスペースに統合し、AIがシステム全体を俯瞰できるようにします。サブプロジェクトごとの設定や秘匿情報の隠蔽状態も、起動時に専用スクリプトで自動検証されます。

## DockMCPによるクロスコンテナ操作
サンドボックス化の代償である「他コンテナへのアクセス不可」を、DockMCPが解消します。
セキュリティポリシーに基づき、AIに**「他のコンテナのログを読み、テストを実行する権限」**を付与。APIとフロントエンドの連携不具合なども、AIがシステム全体を横断して自律的に調査可能になります。



## この環境が解決していない課題

**ネットワーク制限** 

AIが任意の外部ドメインにアクセスするのを防止したい場合、ホワイトリスト方式のプロキシの導入が考えられます。

参考ドキュメント：
- [Docker Compose ネットワーク設定](https://docs.docker.com/compose/networking/)
- [Squid プロキシ](http://www.squid-cache.org/Doc/)

> 参考: [Anthropic公式のサンドボックス環境](https://github.com/anthropics/claude-code/tree/main/.devcontainer) にもファイアウォール設定の例があります。




# ユースケース

### 1. マイクロサービス開発
```
workspace/
├── mobile-app/     ← Flutter/React Native
├── api-gateway/    ← Node.js
├── auth-service/   ← Go
└── db-admin/       ← Python
```

APIキーを公開せずに、AIがすべてのサービスを横断してサポート。

### 2. フルスタックプロジェクト
```
workspace/
├── frontend/       ← React
├── backend/        ← Django
└── workers/        ← Celeryタスク
```

AIがフロントエンドのコードを編集しながら、バックエンドのログを確認可能。

### 3. レガシー + 新規
```
workspace/
├── legacy-php/     ← 古いコードベース
└── new-service/    ← モダンな書き直し
```

AIが両方を理解し、移行を支援。

## 前提条件

### Sandbox： セキュアなサンドボックス + 秘匿情報隠蔽
- **Docker と Docker Compose**（Docker Desktop または OrbStack）
- **VS Code** と **[Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)** 拡張機能

### Sandbox + DockMCP： セキュアなサンドボックス + 秘匿情報隠蔽の環境から AI のクロスコンテナアクセス
- 上記に加えて:
- **DockMCP** - 以下のいずれかの方法でインストール:
  - [GitHub Releases](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases)からビルド済みバイナリをダウンロード
  - ソースからビルド（DevContainer内でビルド可能）
- MCP対応のAIアシスタントCLI（例：`claude` CLI、Gemini（Agentモード））

## アーキテクチャ概要

```
ホストOS
├── DockMCP Server (:8080)
│   ├── AI用 HTTP/SSE API
│   ├── セキュリティポリシー実施
│   └── コンテナアクセスゲートウェイ
│
└── Docker Engine
    ├── DevContainer (AI環境)
    │   ├── Claude Code / Gemini
    │   └── secrets/ → 空（隠蔽）
    │
    ├── API Container
    │   └── secrets/ → 実ファイル
    │
    └── Web Container
```

**データフロー:** AI (DevContainer) → DockMCP (:8080) → 他のコンテナ

### なぜ秘匿ファイルを隠せるのか？

**ポイント:** AIがDevContainer内で動作するため、Dockerボリュームマウントで秘匿ファイルを隠せます。

```
ホストOS
├── demo-apps/securenote-api/.env  ← 実体あり
│
├── DevContainer (AI実行環境)
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

### DevContainerによる隔離のメリット

AIがDevContainer内で動作することで、ホストOSのファイルへのアクセスも制限されます。

```
ホストOS
├── /etc/            ← AIからアクセス不可
├── ~/.ssh/          ← AIからアクセス不可
├── ~/Documents/     ← AIからアクセス不可
├── ~/other-project/ ← AIからアクセス不可
├── ~/secret-memo/   ← AIからアクセス不可
│
└── DevContainer
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

> ⚠️ **Git 操作の注意:** DevContainer 内では隠蔽されたファイル（`.env`、`secrets/` 内のファイル）が「削除された」ように見えます。`git commit -a` や `git add .` を実行すると、意図せずファイルの削除をコミットする可能性があります。コミット操作はホスト側で行うか、DevContainer 内では明示的にファイルを指定して `git add` してください。








# クイックスタート

> **💡 日本語環境にする場合:** DevContainer（または cli_sandbox）を開く前に、ホストOS上で以下を実行：
> ```bash
> .sandbox/scripts/init-env-files.sh -i
> ```
> 言語選択で `2) 日本語` を選ぶと、コンテナ内のターミナル出力が日本語になります。
> （コンテナ内からでも実行できます）

### オプションA: Sandbox

秘匿情報隠蔽付きのセキュアなサンドボックスだけが必要な場合：

```bash
# 1. VS Codeで開く
code .

# 2. コンテナで再度開く（Cmd+Shift+P / F1 → "Dev Containers: Reopen in Container"）
```

**これだけです！** AIは `/workspace` 内のコードにアクセスできますが、`.env` と `secrets/` ディレクトリは隠されています。

**保護されているもの:**
- `.env` ファイル → 空としてマウント
- `secrets/` ディレクトリ → 空に見える
- ホストOSのファイル → 全くアクセス不可

### オプションB: Sandbox + DockMCP

AIが他のコンテナのログを確認したりテストを実行したりする必要がある場合は、DockMCPを使用：

#### ステップ1: DockMCPを起動（ホストOS上で）

```bash
# DockMCPをインストール（~/go/bin/ にインストール）
cd dkmcp
make install

# サーバー起動
dkmcp serve --config configs/dkmcp.example.yaml
```

> **注意:** `make build` ではなく `make install` を使用してください。これにより、バイナリがワークスペース（DevContainerからは見えるが動作しない）ではなく `$GOPATH/bin` にインストールされます。

> **重要:** DockMCPサーバーを再起動した場合、SSE接続が切断されるため、AIアシスタント側で再接続が必要です。Claude Codeでは `/mcp` → 「Reconnect」を実行してください。

#### ステップ2: DevContainerを開く

```bash
code .
# Cmd+Shift+P / F1 → "Dev Containers: Reopen in Container"
```

#### ステップ3: Claude CodeをDockMCPに接続

DevContainer内で：

```bash
claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse
```

**VS Codeを再起動**してMCP接続を有効にします。

#### ステップ4（推奨）: カスタムドメイン設定

より現実的な開発体験のために、カスタムドメインを設定します：

**ホストOS上 - `/etc/hosts` に追加:**
```bash
# macOS/Linux
echo "127.0.0.1 securenote.test api.securenote.test" | sudo tee -a /etc/hosts

# Windows（管理者としてメモ帳を実行）
# 編集: C:\Windows\System32\drivers\etc\hosts
# 追加: 127.0.0.1 securenote.test api.securenote.test
```

> **注意:** DevContainerは `docker-compose.yml` の `extra_hosts` により、カスタムドメインを自動的にホストに解決します。コンテナ内での追加設定は不要です。

#### ステップ5（オプション）: デモアプリで試す

> .env や、key ファイルの用意など必要です。詳細は [demo-apps/README.ja.md](demo-apps/README.ja.md) を参照

```bash
# ホストOS上で - デモアプリを起動
cd demo-apps
docker-compose -f docker-compose.demo.yml up -d --build
```

**アクセス:**
- Web: http://securenote.test:8000
- API: http://api.securenote.test:8000/api/health

**DevContainerから（AIがcurlでテスト可能）:**
```bash
curl http://api.securenote.test:8000/api/health
curl http://securenote.test:8000
```

これでAIは以下が可能に：
- ✅ ログ確認: 「securenote-apiのログを見せて」
- ✅ テスト実行: 「securenote-apiでnpm testを実行して」
- ✅ 疎通確認: カスタムドメインでcurlテスト
- 秘匿情報は依然として保護

---

### トラブルシューティング：DockMCP接続

Claude CodeがDockMCPツールを認識しない場合：

1. **DockMCPが実行中か確認**: `curl http://localhost:8080/health`（ホストOS上で）
2. **MCP再接続を試す** - Claude Codeで `/mcp` を実行し、「Reconnect」を選択
3. **VS Codeを完全に再起動**（Cmd+Q / Alt+F4）- Reconnectで解決しない場合

### フォールバック：DevContainer内でdkmcp clientを使用

MCPプロトコルが動作しない場合（Claude CodeやGeminiが接続できない）、フォールバックとしてDevContainer内で `dkmcp client` コマンドを直接使用できます。

> **注意:** `/mcp` で「✔ connected」と表示されていても、MCPツールが「Client not initialized」エラーで失敗することがあります。これはVS Code拡張機能（Claude Code, Gemini Code Assist等）のセッション管理のタイミング問題が原因である可能性があります。この場合：
> 1. まず `/mcp` → 「Reconnect」を試す（簡単な解決策）
> 2. それでも解決しない場合、AIは `dkmcp client` コマンドをフォールバックとして使用
> 3. 最終手段として、VS Codeを完全に再起動して接続を再確立

**セットアップ（初回のみ）:**

DevContainer内でdkmcpをインストール：
```bash
cd /workspace/dkmcp
make install
```

> **注意:** Go環境はデフォルトで有効です。インストール後、イメージサイズを小さくしたい場合は `.devcontainer/devcontainer.json` の `features` ブロックをコメントアウトしてリビルドできます。

**使用方法:**
```bash
# コンテナ一覧
dkmcp client list --url http://host.docker.internal:8080

# ログ取得
dkmcp client logs --url http://host.docker.internal:8080 securenote-api

# コマンド実行
dkmcp client exec --url http://host.docker.internal:8080 securenote-api "npm test"
```





# コマンド

| コマンド | 実行場所 | 説明 |
|---------|---------|------|
| `dkmcp serve` | ホストOS | DockMCPサーバーを起動 |
| `dkmcp list` | ホストOS | アクセス可能なコンテナを一覧表示 |
| `dkmcp client list` | DevContainer | HTTP経由でコンテナ一覧 |
| `dkmcp client logs <container>` | DevContainer | HTTP経由でログ取得 |
| `dkmcp client exec <container> "cmd"` | DevContainer | HTTP経由でコマンド実行 |

> 詳細なコマンドオプションについては [dkmcp/README.ja.md](dkmcp/README.ja.md#cliコマンド) を参照





# プロジェクト構造

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





# セキュリティ機能

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

コンテナ内の機密ファイルブロック（`blocked_paths`）、Claude Code / Gemini 設定からの自動インポートなど、詳細な設定は [dkmcp/README.ja.md「設定リファレンス」](dkmcp/README.ja.md#設定リファレンス) を参照。

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

パスワード、APIキー、Bearerトークン、認証情報付きデータベースURLなどをデフォルトで検出します。設定方法の詳細は [dkmcp/README.ja.md「出力マスキング」](dkmcp/README.ja.md#出力マスキング) を参照。





# デモを試す

### ハンズオン: 秘匿情報隠蔽を体験してみよう

このプロジェクトでは **2つの隠蔽メカニズム** を使い分けています：

| 方法 | 効果 | 用途 |
|------|------|------|
| Docker マウント | ファイル自体が見えない | `.env`、証明書など |
| `.claude/settings.json` | Claude Code がアクセス拒否 | ソースコード内の秘匿情報 |

---

**🔹 方法1: Docker マウントによる隠蔽**

このハンズオンでは、秘匿設定の **正常な状態** と **設定漏れの状態** の両方を体験します。

#### ステップ1: 正常な状態を確認

まず、現在の設定で秘匿ファイルが正しく隠蔽されていることを確認します。

```bash
# DevContainer 内で実行
# iOS アプリの Config ディレクトリを確認（空に見える）
ls -la demo-apps-ios/SecureNote/Config/

# Firebase 設定ファイルを確認（空または存在しない）
cat demo-apps-ios/SecureNote/GoogleService-Info.plist
```

ディレクトリが空、またはファイルの内容が空であれば、正しく隠蔽されています。

#### ステップ2: 設定漏れを体験する

次に、意図的に設定をコメントアウトして、設定漏れの状態を体験します。

1. `.devcontainer/docker-compose.yml` を編集し、iOS 関連の秘匿設定をコメントアウト：

```yaml
    volumes:
      # ...
      # iOS アプリの Firebase 設定ファイルを隠蔽
      # Hide iOS app Firebase config file
      # - /dev/null:/workspace/demo-apps-ios/SecureNote/GoogleService-Info.plist:ro  # ← コメントアウト

    tmpfs:
      # ...
      # iOS アプリの設定ディレクトリを空にする
      # Make iOS app config directory empty
      # - /workspace/demo-apps-ios/SecureNote/Config:ro  # ← コメントアウト
```

2. DevContainer をリビルド：
   - VS Code: `Cmd+Shift+P` → "Dev Containers: Rebuild Container"

#### ステップ3: 起動時の警告を確認

リビルド後、ターミナルに以下のような警告が表示されます：

**警告1: DevContainer と CLI Sandbox の設定差異**
```
⚠️  秘匿設定に差異があります

両方の docker-compose.yml を同期してください:
  📄 /workspace/.devcontainer/docker-compose.yml
  📄 /workspace/cli_sandbox/docker-compose.yml
```

**警告2: .claude/settings.json との同期漏れ**
```
⚠️  以下のファイルが docker-compose.yml に未設定です:

   📄 demo-apps-ios/SecureNote/GoogleService-Info.plist

これらのファイルは .claude/settings.json でブロックされていますが、
docker-compose.yml のボリュームマウントに設定されていません。

対処方法:
  手動で docker-compose.yml を編集する
  または: .sandbox/scripts/sync-secrets.sh を実行
```

> 💡 **ポイント:** 起動時の検証スクリプトが複数のチェックを行い、設定漏れを検出します。これにより、AI がファイルにアクセスする前に問題に気づくことができます。

#### ステップ4: 秘匿情報が見えてしまうことを確認

エラーが出た状態で、秘匿ファイルの内容を確認してみましょう：

```bash
# Config ディレクトリの内容が見える
cat demo-apps-ios/SecureNote/Config/Debug.xcconfig

# Firebase 設定ファイルの内容も見える
cat demo-apps-ios/SecureNote/GoogleService-Info.plist
```

設定漏れにより、本来隠蔽すべきファイルがコンテナ内に露出し、構造的なアクセス制限が効いていない状態です。

#### ステップ5: 設定を元に戻す

コメントアウトを解除し、再度リビルドして正常な状態に戻してください。

> 📝 **まとめ:** Docker マウントによる秘匿設定は、DevContainer と CLI Sandbox の両方で同期する必要があります。設定漏れがあると起動時に検出され、警告が表示されます。

---

**🔹 方法2: .claude/settings.json による制限（保険 + Dockerマウントによる秘匿対象の提案）**

各サブプロジェクトの `.claude/settings.json` にブロック対象ファイルが定義されている場合、2つの効果があります：

  1. **保険**
    - Claude Code がそのファイルを読めなくなる（Docker マウントの設定漏れがあっても保護される）
  2. **Dockerマウントによる秘匿対象の提案**
    - `sync-secrets.sh` がこの定義を読み取り、Docker マウント設定への反映をアシストできる

つまり `.claude/settings.json` が秘匿対象の定義元（真のソース）で、Docker マウントはそこから派生します。

```bash
# 例: Secrets.swift はファイルとして存在するが...
ls demo-apps-ios/SecureNote/Secrets.swift

# Claude Code は読めない（権限エラーになる）
```

**Docker マウントへの同期:**

`.claude/settings.json` の定義を Docker マウントにも反映するには：

```bash
# 対話的に同期（追加するファイルを選択可能）
.sandbox/scripts/sync-secrets.sh

# オプション:
#   1) すべて追加
#   2) 個別確認
#   3) 追加しない
#   4) プレビュー表示（ドライラン）← 設定内容を確認できる
```

> 💡 **おすすめ:** オプション `4` でプレビューを確認してから、`2`で必要なものだけ追加しましょう。

**マージの仕組み:**

```
demo-apps-ios/.claude/settings.json  ─┐
demo-apps/.claude/settings.json      ─┼─→ /workspace/.claude/settings.json
(他のサブプロジェクト)               ─┘     （マージ結果）
```

- **マージ元**: 各サブプロジェクトの `.claude/settings.json`（リポジトリにコミット済み）
- **マージ結果**: `/workspace/.claude/settings.json`（リポジトリには無い）
- **タイミング**: DevContainer 起動時に自動実行

**マージの条件:**

| 状態 | 動作 |
|------|------|
| `/workspace/.claude/settings.json` が存在しない | マージして作成 |
| 存在するが手動変更なし | 再マージ |
| **存在して手動変更あり** | マージせず手動変更を保護 |

> 💡 手動で `/workspace/.claude/settings.json` を編集した場合、次回起動時に上書きされません。元に戻すにはファイルを削除して再起動してください。

```bash
# マージ元を確認（リポジトリにある）
cat demo-apps-ios/.claude/settings.json

# マージ結果を確認（DevContainer 起動時に作成された）
cat /workspace/.claude/settings.json
```

> 📝 マージは `.sandbox/scripts/merge-claude-settings.sh` で行われます。

---

### デモシナリオ1: 秘匿情報の隔離

```bash
# DevContainer内から（AIが試しても失敗する）
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

### デモシナリオ2: クロスコンテナ開発

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

### デモシナリオ3: マルチプロジェクトワークスペース

このワークスペースには以下が含まれます：
- **バックエンドAPI** (demo-apps/securenote-api)
- **Webフロントエンド** (demo-apps/securenote-web)
- **iOSアプリ** (demo-apps-ios/)

Claude Codeができること：
- すべてのソースコードを見る（アプリとサーバー間の連携不具合の調査が可能）
- 任意のコンテナのログを確認（DockMCP経由）
- プロジェクト横断でテストを実行
- クロスコンテナの問題をデバッグ






# 2つの環境

| 環境 | 用途 | 使用タイミング |
|-------------|---------|-------------|
| **DevContainer** (`.devcontainer/`) | VS Codeでの主要開発 | 日常的な開発 |
| **CLI Sandbox** (`cli_sandbox/`) | 代替/復旧 | DevContainerが壊れた時 |

### なぜ2つの環境？

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






# 高度な使い方

### プラグインの活用（マルチリポ構成）

マルチリポ構成（各プロジェクトが独立したGitリポジトリ）でClaude Codeプラグインを使う場合は工夫が必要です。詳細は [プラグインガイド](docs/plugins.ja.md) を参照。

> **注意**: このセクションは Claude Code 専用です。Gemini Code Assist では使えません。

### カスタムDockMCP設定

```yaml
# dkmcp.yaml
security:
  mode: "strict"  # 読み取り専用（logs, inspect, stats）

  allowed_containers:
    - "prod-*"      # 本番コンテナのみ

  exec_whitelist: {}  # コマンド実行なし
```

複数インスタンスの起動など、詳細は [dkmcp/README.ja.md「サーバー起動」](dkmcp/README.ja.md#複数インスタンスの起動) を参照。

### プロジェクト名のカスタマイズ

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

### 起動時出力オプション

DevContainer と CLI Sandbox は起動時に検証スクリプトを実行します。出力量を制御できます：

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

### 同期警告からのファイル除外

起動スクリプトは `.claude/settings.json` でブロックされたファイルが `docker-compose.yml` でも隠蔽されているかチェックします。特定のパターン（`.example` ファイルなど）を警告から除外するには、`.sandbox/config/sync-ignore` を編集します：

```gitignore
# example/template ファイルを同期警告から除外
**/*.example
**/*.sample
**/*.template
```

これは gitignore 形式のパターンを使用します。これらのパターンにマッチするファイルは「docker-compose.yml に未設定」警告をトリガーしません。

### 複数のDevContainerを起動する場合

完全に分離したDevContainer環境が必要な場合（例：異なるクライアント案件）、`COMPOSE_PROJECT_NAME` を使って分離したインスタンスを作成できます。

#### 方法A: .env ファイルで分離（推奨）

`.devcontainer/.env` で異なるプロジェクト名を設定：

```bash
COMPOSE_PROJECT_NAME=client-a
```

別のワークスペースでは：

```bash
COMPOSE_PROJECT_NAME=client-b
```

#### 方法B: コマンドラインで分離

異なるプロジェクト名でDevContainerを起動：

```bash
# プロジェクトA
COMPOSE_PROJECT_NAME=client-a docker-compose up -d

# プロジェクトB（別のボリュームが作成される）
COMPOSE_PROJECT_NAME=client-b docker-compose up -d
```

> ⚠️ **注意:** プロジェクト名が異なるとボリュームも別になるため、ホームディレクトリ（認証情報・設定・履歴）は自動的に共有されません。下記「ホームディレクトリのコピー」を参照。

#### 方法C: バインドマウントでホームディレクトリを共有

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

#### ホームディレクトリのエクスポート/インポート

ホームディレクトリ（認証情報・設定・履歴）をバックアップまたは別のワークスペースに移行できます：

```bash
# ワークスペース全体をエクスポート（devcontainer と cli_sandbox の両方）
./.sandbox/scripts/copy-credentials.sh --export /path/to/workspace ~/backup

# 特定の docker-compose.yml からエクスポート
./.sandbox/scripts/copy-credentials.sh --export .devcontainer/docker-compose.yml ~/backup

# ワークスペースにインポート
./.sandbox/scripts/copy-credentials.sh --import ~/backup /path/to/workspace
```

**注意:** インポート先のボリュームが存在しない場合、先に環境を一度起動してボリュームを作成する必要があります。

用途：
- `~/.claude/` の使用量データを確認
- 設定のバックアップ
- 新しいワークスペースへの認証情報の移行
- トラブルシューティング






# 自分のプロジェクトへの適用

このリポジトリは **GitHub テンプレートリポジトリ** として設計されています。テンプレートから自分のプロジェクトを作成できます。

### テンプレートとして使う

#### Step 1: テンプレートから作成

GitHub で **「Use this template」** → **「Create a new repository」** をクリック。

作成されるリポジトリの特徴：
- テンプレートの全ファイル（このリポジトリのコミット履歴なし）
- 新しいGit履歴からスタート
- アップストリームとは独立（自動同期なし）

#### Step 2: 新しいリポジトリをクローン

```bash
git clone https://github.com/your-username/your-new-repo.git
cd your-new-repo
```

#### 更新をチェック

テンプレートから作成したリポジトリはアップストリームの更新を自動で受け取れないため、**更新通知機能** を搭載しています。

**仕組み:**
- DevContainer 起動時に GitHub の新しいリリースをチェック
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

**デバッグ:** 更新チェックの内部動作を確認するには `--debug` を使います：
```bash
.sandbox/scripts/check-upstream-updates.sh --debug
```

---

### 別の方法: 直接クローン

Git で上流の変更を追跡したい場合（コントリビュート目的など）：

```bash
git clone https://github.com/YujiSuzuki/ai-sandbox-dkmcp.git
cd ai-sandbox-dkmcp
```

---

### プロジェクトのカスタマイズ

テンプレートを使用した場合も直接クローンした場合も、以下の手順で環境をカスタマイズします。

#### demo-apps を自分のプロジェクトに置き換え

```bash
# デモアプリを削除（または参考用に残す）
rm -rf demo-apps demo-apps-ios

# 自分のプロジェクトを配置
git clone https://github.com/your-org/your-api.git
git clone https://github.com/your-org/your-web.git
```

#### 秘匿ファイルの隠蔽設定

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

#### DockMCP設定

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

#### DevContainerをリビルド

```bash
# VS Code で Command Palette を開く (Cmd/Ctrl + Shift + P)
# "Dev Containers: Rebuild Container" を実行
```

#### 動作確認

```bash
# DevContainer内で秘匿ファイルが隠されていることを確認
cat your-api/.env
# → 空または "No such file"

# DockMCPでコンテナにアクセスできることを確認
# Claude Code に "コンテナ一覧を見せて" と聞く
# Claude Code に "your-apiのログを見せて" と聞く
```

#### チェックリスト

- [ ] `.devcontainer/docker-compose.yml` で秘匿ファイルを設定
- [ ] `cli_sandbox/docker-compose.yml` で同じ設定を適用
- [ ] `dkmcp.yaml` でコンテナ名を設定
- [ ] `dkmcp.yaml` で許可コマンドを設定
- [ ] DevContainerをリビルド
- [ ] 秘匿ファイルが隠されていることを確認
- [ ] DockMCP経由でログ確認できることを確認





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
A: **いいえ、推奨しません。** DockMCPには認証機能がないため、ローカル開発環境での使用を想定しています。本番環境やインターネットに公開されたサーバーでの使用は避けてください。使用する場合は自己責任でお願いします。

**Q: 別の秘匿情報管理を使えますか？**
A: はい！他の秘匿情報管理の方法とも組み合わせられます。

**Q: Windowsでも動作しますか？**
A: はい。Docker Desktop があれば Windows/macOS/Linux で動作します。





# ドキュメント

- [DockMCP ドキュメント](dkmcp/README.ja.md) - MCPサーバーのセットアップと使用方法
- [DockMCP 設計思想](dkmcp/README.ja.md#設計思想) - なぜDockMCPはコンテナライフサイクル操作をサポートしないのか
- [プラグインガイド](docs/plugins.ja.md) - マルチリポ構成でのClaude Codeプラグイン活用
- [デモアプリガイド](demo-apps/README.ja.md) - SecureNoteデモの実行方法
- [CLI Sandbox ガイド](cli_sandbox/README.ja.md) - ターミナルベースのサンドボックス
- [CLAUDE.md](CLAUDE.md) - AIアシスタント向けの説明

## ライセンス

MIT License - [LICENSE](LICENSE) を参照
