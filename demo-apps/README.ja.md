# SecureNote デモアプリケーション

**DockMCP**のデモンストレーションアプリケーション - AIコーディングアシスタントを使いながら秘密情報を保護する方法を体験できます。

[English README is here](README.md)

> ⚠️ **注意:** このデモは動作検証が十分ではありません。不具合を見つけたら、DockMCPを使ってAIに調査してもらうことができます。それ自体がDockMCPの実践的な使い方です。

## このデモが示すもの

### 問題
通常、DevContainer内でAIアシスタント（Claude Code、Gemini Code Assist）を使用すると:
- AIはコンテナにマウントされたすべてのファイルを見ることができる
- 機密ファイル（APIキー、暗号化キー、`.env`）が露出する
- AIの学習データに誤って漏洩するリスク

### 解決策
**このSandbox環境**と**DockMCP**の組み合わせが提供:
1. **秘密情報の隔離** - ボリュームマウントを使ってAIから機密ファイルを隠す
2. **クロスコンテナアクセス** - AIはDockMCP経由でログ確認、テスト実行が可能
3. **通常の開発** - ワークフローの中断なし

## アーキテクチャ

```
┌──────────────────────────────────────────┐
│ DevContainer (AI環境)                    │
│                                          │
│ secrets/ → 空（tmpfs）     🔐 隠蔽       │
│ .env → /dev/null           🔐 隠蔽       │
│                                          │
│ Claude Codeができること:                 │
│ ✅ アプリケーションコードを読む           │
│ ✅ DockMCPでAPIログを確認              │
│ ✅ DockMCPでテスト実行                 │
│ 🔐 秘密情報は読めない                    │
└──────────────────────────────────────────┘

┌──────────────────────────────────────────┐
│ API Container (プロジェクト実行環境)      │
│                                          │
│ secrets/ → 実ファイル      ✅ 見える     │
│ .env → 実際の設定          ✅ 見える     │
│                                          │
│ APIは完全アクセスで正常動作              │
└──────────────────────────────────────────┘
```

## クイックスタート

### オプション1: Webデモ（最初のお試しに推奨）

**所要時間:** 5分
**必要なもの:** Docker Desktop（または OrbStack）

```bash
# 1. デモアプリケーションを起動
cd demo-apps
docker-compose -f docker-compose.demo.yml up -d

# 2. ログを確認して起動完了を待つ（約30秒）
#    "Server running on port 8080" などが表示されたら Ctrl+C で終了
docker-compose -f docker-compose.demo.yml logs -f

# 3. /etc/hosts にカスタムドメインを追加（初回のみ）
echo "127.0.0.1 securenote.test api.securenote.test" | sudo tee -a /etc/hosts

# 4. ブラウザで開く
open http://securenote.test:8000
```

> **注意:** nginx設定によりドメイン名でのアクセスが必要です。`localhost:8000` では404になります。

**ログイン:**
- ユーザー名: `demo` パスワード: `demo123`
- ユーザー名: `alice` パスワード: `alice123`

**試してみる:**
1. デモ認証情報でログイン
2. 暗号化されたメモを作成
3. メモはAIが見えない秘密情報を使って暗号化される！

### オプション2: DockMCPでプロジェクトのコンテナへ安全にアクセス

**所要時間:** 15分
**必要なもの:** Docker Desktop（または OrbStack） + DockMCP

```bash
# 1. ホストOSでDockMCPをインストール・起動
cd ../dkmcp
make install  # ~/go/bin/ にインストール（初回のみ）
dkmcp serve --config configs/dkmcp.example.yaml
# DockMCPが http://localhost:8080 で動作

# 2. デモアプリケーションを起動
cd ../demo-apps
docker-compose -f docker-compose.demo.yml up -d

# 3. VS CodeでDevContainerを開く
code ..
# Claude Codeが自動的にDockMCPに接続

# 4. Claude Codeに聞いてみる:
"securenote-apiのログを表示して"
"securenote-apiコンテナでテストを実行して"
"APIで秘密情報が読み込まれているか確認して"
```

## プロジェクト構造

```
demo-apps/
├── securenote-api/          # バックエンドAPI (Node.js)
│   ├── src/
│   │   ├── server.js
│   │   ├── routes/
│   │   │   ├── auth.js      # JWT認証
│   │   │   ├── notes.js     # 暗号化付きCRUD
│   │   │   └── demo.js      # 秘密情報ステータスエンドポイント
│   │   ├── services/
│   │   │   └── encryption.js
│   │   └── middleware/
│   ├── secrets/             # 🔒 AIから隠蔽
│   │   ├── jwt-secret.key
│   │   └── encryption.key
│   ├── .env                 # 🔒 AIから隠蔽
│   └── tests/
│
├── securenote-web/          # Webフロントエンド (React + Vite)
│   ├── src/
│   │   ├── App.jsx
│   │   ├── pages/
│   │   ├── components/
│   │   └── services/
│   └── Dockerfile
│
└── docker-compose.demo.yml  # デモオーケストレーション
```

## APIエンドポイント

### 認証
- `POST /api/auth/login` - ユーザー名/パスワードでログイン

### ノート（認証必要）
- `GET /api/notes` - すべてのノートを一覧表示（復号化済み）
- `GET /api/notes/:id` - 特定のノートを取得
- `POST /api/notes` - 新しいノートを作成（暗号化）
- `PUT /api/notes/:id` - ノートを更新
- `DELETE /api/notes/:id` - ノートを削除

### デモ
- `GET /api/health` - ヘルスチェック
- `GET /api/demo/secrets-status` - 秘密情報が読み込まれているか確認

## 秘密情報隔離のテスト

### DevContainer（AI Sandbox環境）から:

```bash
# 秘密情報を読もうとする
cat demo-apps/securenote-api/secrets/jwt-secret.key
# 出力: (空またはエラー)

cat demo-apps/securenote-api/.env
# 出力: (空)

# しかしDockMCPは使える！
# Claude Codeに聞く: "APIのログを確認して"
# Claude Codeに聞く: "securenote-apiで npm test を実行して"
```

### APIが秘密情報を持っていることを確認:

```bash
# デモエンドポイントを呼び出す
curl http://api.securenote.test:8000/api/demo/secrets-status

# レスポンス:
{
  "message": "This API has access to secrets",
  "secretsLoaded": true,
  "proof": {
    "jwtSecretLoaded": true,
    "jwtSecretPreview": "super-sec***",
    "encryptionKeyLoaded": true
  }
}
```

## デモを停止

```bash
cd demo-apps
docker-compose -f docker-compose.demo.yml down
```

## アプリケーションへのアクセス

| アプリケーション | URL |
|---|---|
| **Web版** | http://securenote.test:8000 |
| **API** | http://api.securenote.test:8000 |

> `/etc/hosts` への追加が必要です（クイックスタート参照）

## 詳細情報

- [DockMCP ドキュメント](../dkmcp/README.ja.md)
- [AI Sandbox Environment](../README.ja.md)
- [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)
