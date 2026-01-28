# SecureNote API

SecureNote デモアプリケーションのバックエンド API です。AI コーディングアシスタントと連携する際の、安全なシークレット管理の手法を示しています。

## 機能

- 🔐 **暗号化ストレージ** - ノートの内容は保存時に暗号化されます
- 🔑 **JWT 認証** - トークンベースの認証方式
- 🚫 **シークレット分離** - DevContainer 内の AI アシスタントからシークレットを隠蔽
- ✅ **完全動作** - シークレットが隠されていても API は正常に動作します

## API エンドポイント

### 認証

- `POST /api/auth/login` - ログインして JWT トークンを取得

### ノート

- `GET /api/notes` - 全ノートを一覧表示（復号化済み）
- `GET /api/notes/:id` - 特定のノートを取得
- `POST /api/notes` - 新規ノートを作成（内容は暗号化されます）
- `PUT /api/notes/:id` - ノートを更新
- `DELETE /api/notes/:id` - ノートを削除

### デモ用

- `GET /api/demo/secrets-status` - シークレットが読み込まれているか確認（デモの検証用）
- `GET /api/health` - ヘルスチェック

## デモ用ユーザー

| ユーザー名 | パスワード |
|------------|------------|
| demo       | demo123    |
| alice      | alice123   |

## ローカル開発

```bash
# 依存パッケージをインストール
npm install

# 環境ファイルをコピー
cp .env.example .env

# サーバーを起動
npm start

# テストを実行
npm test
```

## Docker での実行

```bash
# ビルド
docker build -t securenote-api .

# 実行
docker run -p 8080:8080 \
  -v $(pwd)/secrets:/app/secrets:ro \
  -v $(pwd)/.env:/app/.env:ro \
  securenote-api
```

## セキュリティデモ

### AI から隠されるもの

DevContainer 内で動作する場合、以下のファイルは AI アシスタントから見えません：
- `secrets/jwt-secret.key`
- `secrets/encryption.key`
- `.env`

### 仕組み

1. DevContainer は `secrets/` ディレクトリを空の tmpfs としてマウント
2. DevContainer は `.env` を `/dev/null` としてマウント
3. AI はシークレットを読み取れない
4. 一方、API コンテナは実際のシークレットにアクセス可能
5. 開発は通常通り続けられます

### 検証方法

```bash
# DevContainer（AI 環境）から
cat demo-apps/securenote-api/secrets/jwt-secret.key
# => （空または エラー）

# DockMCP を使って API に問い合わせ
curl http://api.securenote.test:8000/api/demo/secrets-status
# => { "secretsLoaded": true, "proof": { ... } }
```

これにより、API にはシークレットがあるが、AI からは見えないことが確認できます。

## アーキテクチャ

```
┌─────────────────────────────┐
│ DevContainer (AI)           │
│ secrets/ → (空の tmpfs)     │  🔐 AI は読み取れない
│ .env → /dev/null            │
└─────────────────────────────┘

┌─────────────────────────────┐
│ API コンテナ                │
│ secrets/ → 実際のファイル   │  ✅ API は読み取り可能
│ .env → 実際の設定           │
└─────────────────────────────┘
```

## DockMCP 経由のテスト

DockMCP MCP サーバーが動作していれば、AI アシスタントは以下のような操作が可能です：

```javascript
// コンテナ一覧を取得
dkmcp.list_containers()

// API のログを取得
dkmcp.get_logs("securenote-api", { tail: "50" })

// テストを実行
dkmcp.exec_command("securenote-api", "npm test")

// リソース統計を確認
dkmcp.get_stats("securenote-api")
```

シークレットに直接アクセスせずに、これらの操作ができます。
