# 会話履歴検索ツール (search-history)

Claude Code の過去の会話を検索・閲覧できるツールです。

AI Sandbox 内で AI に話しかけるだけで使えます。
SandboxMCP がツールを自動検出し、AI が必要に応じて実行します。

## できること

### 1. キーワード検索

過去の会話から特定のキーワードを検索します。

```
あなた: 「DockMCP の設定について話したの覚えてる？」
AI:     search-history で "DockMCP" を検索 → 該当する会話を要約して回答
```

### 2. セッション一覧

期間を指定して、どんなセッションがあったかを確認できます。

```
あなた: 「先週の会話の一覧見せて」
AI:     -list -after 2026-02-03 -before 2026-02-07 → セッション一覧を表示
```

出力例:
```
00de1fea-a91  02/04 14:32 ~ 02/09   1179 msgs  3409KB  コンテナ一覧みせて
1aef51ad-11d  02/05 14:05              163 msgs   542KB  comparison-article.md
494c72df-149  02/03 14:39 ~ 02/04     506 msgs  8102KB  README.ja.md
```

### 3. 活動の振り返り

日ごと・週ごとの作業内容を AI にまとめてもらえます。

```
あなた: 「昨日なにやったっけ？」
AI:     昨日のメッセージを検索 → 日ごとのトピックを要約

あなた: 「先週の概要もらえる？」
AI:     1週間分のセッションを調べ → 日別の活動概要を作成
```

### 4. セッションの閲覧

特定のセッションの中身を時系列で確認できます。

```
あなた: 「さっきの会話の詳細見たい」
AI:     -session <id> で該当セッションのメッセージを表示
```

### 5. 事後調査（フォレンジック）

身に覚えのないファイルや変更が見つかったとき、過去のAIセッションを遡って原因を特定できます。

AIセッションは使い捨てなので、前のセッションで何が起きたかは通常わかりません。会話履歴だけが唯一の記録であり、このツールがそれを引き出します。

```
あなた: 「この search-history バイナリ、いつ誰が作ったの？」
AI:     Bash の実行履歴を検索
        → 過去のセッションで go build が -o なしで実行された経緯を特定
```

**実際にあった例:**

コードレビューで `.sandbox/sandbox-mcp/search-history`（3.5MB のバイナリ）がステージングされているのを発見。git log には記録がなく、ファイルのタイムスタンプだけでは経緯が不明。search-history で Bash の実行履歴を検索したところ、別のセッションで AI がコンパイル確認のために `go build` を `-o` 指定なしで実行し、バイナリが残っていたことが判明しました。

## コマンドラインでの使い方

AI に頼まなくても、直接実行できます。

### 3つのモード

| モード | コマンド | 説明 |
|--------|----------|------|
| キーワード検索 | `go run .sandbox/tools/search-history.go "検索語"` | 正規表現にも対応 |
| セッション一覧 | `go run .sandbox/tools/search-history.go -list` | 最終活動日順 |
| セッション閲覧 | `go run .sandbox/tools/search-history.go -session <id>` | ID は先頭数文字でOK |

### フィルタオプション

| オプション | 説明 | 例 |
|------------|------|----|
| `-role <role>` | ロールで絞り込み | `-role user`（自分の発言のみ） |
| `-tool <name>` | ツール名で絞り込み（`-role tool` と併用） | `-role tool -tool Bash` |
| `-after <date>` | 指定日以降 | `-after 2026-02-01` |
| `-before <date>` | 指定日以前 | `-before 2026-02-07` |
| `-i` | 大文字小文字を無視 | `-i "dockmcp"` |
| `-project <name>` | プロジェクト指定（デフォルト: workspace） | `-project all` |

### 表示オプション

| オプション | 説明 | デフォルト |
|------------|------|------------|
| `-max <n>` | 最大表示件数 | 50（0 = 無制限） |
| `-context <n>` | 1エントリの表示文字数 | 200（0 = 全文） |
| `-no-color` | カラー出力を無効化 | — |

### 使用例

```bash
# キーワード検索
go run .sandbox/tools/search-history.go "DockMCP"

# 自分の発言だけ検索
go run .sandbox/tools/search-history.go -role user "docker"

# Bash ツールの実行履歴から検索
go run .sandbox/tools/search-history.go -role tool -tool Bash "git"

# 日付を絞って検索
go run .sandbox/tools/search-history.go -after 2026-01-20 "secret"

# 全プロジェクトを横断検索
go run .sandbox/tools/search-history.go -project all "error"

# 特定の日のセッション一覧
go run .sandbox/tools/search-history.go -list -after 2026-02-08 -before 2026-02-08

# セッションの全文を表示（文字数制限なし・件数制限なし）
go run .sandbox/tools/search-history.go -session c01514d6 -context 0 -max 0
```

## 日付フィルタの動作

`-after` / `-before` は、セッションの開始日ではなく、**メッセージのタイムスタンプ**で判定します。

- キーワード検索: 指定期間内のメッセージだけがヒット
- セッション一覧 (`-list`): 指定期間にメッセージがあるセッションを表示
- 複数日にまたがるセッションでも、該当日にメッセージがあれば表示される

日付はローカルタイムゾーンで解釈されます。

## 仕組み

Claude Code は会話履歴を `~/.claude/projects/` 以下に JSONL 形式で保存しています。search-history はこのファイルを直接読み取って検索します。

```
~/.claude/projects/<project-dir>/
  └── <session-id>.jsonl    ← 会話データ
```

SandboxMCP 経由で AI が `run_tool` として実行する場合も、同じファイルを読みに行きます。
