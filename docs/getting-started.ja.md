# はじめてのセットアップガイド

ゼロから AI Sandbox + DockMCP が動く状態までを、一歩ずつ案内します。

[← README に戻る](../README.ja.md)

---

## このガイドの対象

- Docker や VS Code は使ったことがあるが、このプロジェクトは初めて
- AI Sandbox と DockMCP の関係をざっくり理解したい
- まず動かしてみたい

**所要時間の目安:** 15〜30 分（DockMCP なしなら 5 分）

---

## 全体像

セットアップは3段階あります。必要な範囲だけ進めてください。

```
ステップ 1〜3: AI Sandbox を動かす（必須）
    ↓
ステップ 4〜6: DockMCP を接続する（推奨）
    ↓
ステップ 7〜8: デモアプリで試す（任意）
```

| 段階 | できること |
|------|-----------|
| **Sandbox のみ** | AIがコードを読み書きできる。秘匿ファイルは隠されている |
| **+ DockMCP** | AIが他コンテナのログ確認・テスト実行もできる |
| **+ デモアプリ** | 付属のデモで一通りの機能を体験できる |

---

## ステップ 1: 前提条件を確認する

以下がインストールされていることを確認してください。

| ツール | 確認コマンド | インストール先 |
|--------|-------------|----------------|
| **Docker** | `docker --version` | [Docker Desktop](https://www.docker.com/products/docker-desktop/) または [OrbStack](https://orbstack.dev/) |
| **VS Code** | `code --version` | [Visual Studio Code](https://code.visualstudio.com/) |
| **Dev Containers 拡張** | VS Code の拡張機能で確認 | [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) |

DockMCP も使う場合は追加で必要：

| ツール | 確認コマンド | インストール先 |
|--------|-------------|----------------|
| **Go** (1.24 以上) | `go version` | [go.dev](https://go.dev/dl/) |

> [!TIP]
> **Go がなくても大丈夫:** AI Sandbox の中に Go 環境があるため、ホスト向けバイナリをクロスビルドできます（ステップ 4 で説明）。

### 期待される結果

```bash
$ docker --version
Docker version 27.x.x, build xxxxxxx   # バージョンが表示されれば OK

$ code --version
1.9x.x                                  # バージョンが表示されれば OK
```

Docker Desktop（または OrbStack）が**起動中**であることも確認してください。

---

## ステップ 2: リポジトリを取得する

```bash
# 方法A: テンプレートから（GitHub で "Use this template" → クローン）
git clone https://github.com/your-username/your-new-repo.git
cd your-new-repo

# 方法B: 直接クローン
git clone https://github.com/YujiSuzuki/ai-sandbox-dkmcp.git
cd ai-sandbox-dkmcp
```

> [!TIP]
> 詳しい選択肢は [自分のプロジェクトへの適用](customization.ja.md) を参照。

### （オプション）日本語環境にする

DevContainer を開く前に、ホスト OS 上で以下を実行しておくと、コンテナ内のターミナル出力が日本語になります。

```bash
.sandbox/scripts/init-host-env.sh -i
# → 言語選択で「2) 日本語」を選ぶ
```

### 期待される結果

```
your-repo/
├── .devcontainer/
├── .sandbox/
├── dkmcp/
├── demo-apps/
└── README.md
```

上記のようなディレクトリ構造が見えれば OK。

---

## ステップ 3: DevContainer を起動する

```bash
code .
```

VS Code が開いたら：

1. 右下に **「Reopen in Container」** の通知が出る → クリック
2. 通知が出ない場合 → `Cmd+Shift+P`（macOS）/ `Ctrl+Shift+P`（Windows/Linux）→ **「Dev Containers: Reopen in Container」**

初回はコンテナのビルドが走るため、数分かかります。

### 初回起動で何が起きるか

DevContainer の起動中、以下の処理が自動で行われます：

1. Docker イメージのビルド（初回のみ、2回目以降はキャッシュ利用）
2. コンテナの起動
3. AI 設定のマージ（サブプロジェクトの `.claude/settings.json` を統合）
4. 秘匿設定の検証（`docker-compose.yml` の設定が正しいか自動チェック）
5. SandboxMCP のビルドと登録（`.sandbox/` 内のツールを AI が使えるようにする）
6. テンプレートの更新チェック

VS Code のターミナルに検証結果が表示されます。`✓`（成功）が並んでいれば問題ありません。

### 期待される結果

- VS Code の左下に **「Dev Container: AI Sandbox」** と表示される
- ターミナルが開き、`/workspace` ディレクトリにいる
- `ls` でプロジェクトのファイルが見える

```bash
$ ls demo-apps/securenote-api/.env
demo-apps/securenote-api/.env     # ファイルは存在するように見えるが…

$ cat demo-apps/securenote-api/.env
                                  # 中身は空！（隠蔽されている）
```

**ここまでで AI Sandbox は使える状態です。** Claude Code や Gemini Code Assist を起動して、コードの読み書きを試してみてください。

DockMCP（他コンテナへのアクセス）が不要なら、[次のステップ](#次のステップ) へスキップできます。

> [!TIP]
> **VS Code を使わない場合:** ターミナルだけで作業したい場合は、CLI Sandbox（`cli_sandbox/`）も利用できます。`./cli_sandbox/claude.sh` で Claude Code を、`./cli_sandbox/gemini.sh` で Gemini CLI を起動できます。詳しくは [リファレンス](reference.ja.md) を参照してください。

---

## ステップ 4: DockMCP をビルドする（ホスト OS 上で）

> [!IMPORTANT]
> ここからはホスト OS 上（DevContainer の外）で作業します。VS Code のターミナルではなく、別のターミナルウィンドウを開いてください。

```bash
cd dkmcp
make install
```

これで `dkmcp` コマンドが `~/go/bin/` にインストールされます。

<details>
<summary>ホスト OS に Go がない場合</summary>

AI Sandbox の中には Go 環境があるので、ホスト OS 向けのバイナリをクロスビルドできます。

**AI Sandbox 内で実行：**
```bash
cd /workspace/dkmcp
make build-host
```

**ホスト OS で実行：**
```bash
cd <リポジトリのパス>/dkmcp
make install-host DEST=~/go/bin        # Go がある場合
make install-host DEST=/usr/local/bin  # Go がない場合
```

</details>

### 期待される結果

```bash
$ dkmcp version
x.x.x    # バージョンが表示されれば OK
```

---

## ステップ 5: DockMCP サーバーを起動する（ホスト OS 上で）

引き続きホスト OS 上で：

```bash
dkmcp serve --config configs/dkmcp.example.yaml
```

`--sync` を付けると、`.sandbox/host-tools/` にあるスクリプト（デモアプリのビルド・起動・停止など）を AI が DockMCP 経由で実行できるようになります。初回実行時に AI がホストツールを使おうとすると、ユーザーに承認を求めるプロンプトが表示されます。

```bash
dkmcp serve --config configs/dkmcp.example.yaml --sync
```

### 期待される結果

```
DockMCP server started on :8080
Security mode: moderate
Allowed containers: securenote-*, demo-*
```

このターミナルは開いたままにしてください（サーバーが動き続けます）。

### 接続確認（ホスト OS の別ターミナルで）

```bash
curl http://localhost:8080/health
# → 200 OK が返れば成功
```

---

## ステップ 6: AI Sandbox から DockMCP に接続する

**VS Code の DevContainer 内のターミナル**に戻って：

```bash
# Claude Code の場合
claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse

# Gemini CLI の場合
gemini mcp add --transport sse dkmcp http://host.docker.internal:8080/sse
```

登録後、接続を有効にします：

- **Claude Code:** `/mcp` と入力 → 「Reconnect」を選択
- **VS Code 全体を再起動**（`Cmd+Q` / `Alt+F4` → 再度開く）でも OK

### 期待される結果

Claude Code で `/mcp` を実行すると、`dkmcp` が **connected** と表示される：

```
  dkmcp
  ✔ connected
  17 tools
```

試しに AI に話しかけてみてください：

```
「コンテナの一覧を見せて」
```

> [!NOTE]
> デモアプリをまだ起動していない場合、コンテナは空かもしれません。それでも接続自体が確認できれば OK です。

### うまくいかない場合

- [トラブルシューティング](reference.ja.md#トラブルシューティング) を参照
- DockMCP サーバーが起動しているか確認（ステップ 5）
- VS Code のポートパネルで 8080 がフォワードされていたら停止

---

## ステップ 7: デモアプリを起動する（任意）

DockMCP の機能をフルに体験するには、付属のデモアプリを起動します。

**ホスト OS 上で：**

```bash
cd demo-apps
docker compose -f docker-compose.demo.yml up -d --build
```

ステップ 5 で `--sync` を付けて起動していれば、AI に頼むこともできます：
```
「デモアプリをビルドして起動して」
```
> [!NOTE]
> AI がホストツールを初めて使う際、Claude Code が承認を求めるダイアログを表示します。許可すると以降は自動で実行されます。

### （推奨）カスタムドメインの設定

ブラウザでデモアプリにアクセスしやすくなります。

```bash
# ホスト OS 上で実行
echo "127.0.0.1 securenote.test api.securenote.test" | sudo tee -a /etc/hosts
```

### 期待される結果

```bash
# ホスト OS でコンテナ確認
$ docker ps
CONTAINER ID   IMAGE              STATUS    NAMES
xxxxxxxxxxxx   securenote-api     Up        securenote-api
xxxxxxxxxxxx   securenote-web     Up        securenote-web
```

ブラウザでアクセス：
- Web: http://securenote.test:8000（ドメイン設定済みの場合）
- API: http://api.securenote.test:8000/api/health

---

## ステップ 8: AI に話しかけてみる

AI Sandbox 内の Claude Code（または Gemini）で、以下を試してみてください：

### 基本操作

```
「securenote-api のログを見せて」
→ DockMCP 経由でコンテナログが表示される

「securenote-api で npm test を実行して」
→ テスト結果が返ってくる

「使えるスクリプトある？」
→ SandboxMCP 経由で .sandbox/ 内のスクリプト一覧が表示される
```

### セキュリティ確認

```
「demo-apps/securenote-api/.env の中身を見せて」
→ 空のファイル（秘匿情報は隠されている）

「秘匿ファイルがあるか確認してみて」
→ AI が検証スクリプトを実行し、隠蔽状態を報告してくれる
```

### DockMCP の機能

```
「securenote-api コンテナの詳細情報を見せて」
→ inspect 結果が表示される

「securenote-api のメモリ使用量は？」
→ コンテナのリソース統計が表示される
```

---

## 次のステップ

セットアップが完了したら、目的に応じて次のドキュメントに進んでください。

| やりたいこと | ドキュメント |
|-------------|-------------|
| セキュリティ機能を詳しく体験する | [ハンズオン](hands-on.ja.md) |
| 自分のプロジェクトで使う | [自分のプロジェクトへの適用](customization.ja.md) |
| アーキテクチャを理解する | [アーキテクチャ詳細](architecture.ja.md) |
| 他のツールとの違いを知る | [既存ソリューションとの比較](comparison.ja.md) |
| ネットワーク制限を追加する | [ネットワーク制限](network-firewall.ja.md) |

---

## よくあるトラブルと対処法

### 「Reopen in Container」が出てこない

- Dev Containers 拡張がインストールされているか確認
- `Cmd+Shift+P` → 「Dev Containers: Reopen in Container」を手動で実行

### 初回ビルドが遅い

- Docker イメージのダウンロードとビルドに 3〜5 分かかることがあります
- 2 回目以降はキャッシュが効くため速くなります

### DockMCP に接続できない

1. DockMCP サーバーが起動中か確認: `curl http://localhost:8080/health`
2. VS Code のポートパネルで 8080 がフォワードされていたら停止
3. `/mcp` → 「Reconnect」を試す
4. VS Code を完全に再起動（`Cmd+Q` → 再度開く）

詳しくは [トラブルシューティング](reference.ja.md#トラブルシューティング) を参照。

### デモアプリのコンテナが見つからない

- ホスト OS 上で `docker ps` を実行し、コンテナが起動しているか確認
- `docker compose -f docker-compose.demo.yml up -d --build` を再実行
- DockMCP の設定（`dkmcp.example.yaml`）で `allowed_containers` にコンテナ名パターンが含まれているか確認
