---
description: ローカルコードレビューを実行（Git リポジトリがなくても動作）
argument-hint: [project-path]
allowed-tools: [Read, Glob, Grep, Bash(git:*), Bash(ls:*), Bash(find:*), Task, AskUserQuestion, TodoWrite]
---

# ローカルコードレビュー

ローカルのコードに対してコードレビューを実行します。Git リポジトリがある場合はブランチ間の差分を、ない場合は指定されたファイル/ディレクトリをレビューします。

## 引数

ユーザーが指定した引数: $ARGUMENTS

## 実行手順

以下の手順を正確に実行してください：

### Step 1: プロジェクト選択と Git 検出

1. `/workspace` 配下でプロジェクトを検索（Git リポジトリと通常のディレクトリ両方）:
   ```bash
   # Git リポジトリを検索
   find /workspace -name ".git" -type d -maxdepth 3 2>/dev/null | sed 's/\/.git$//'
   # 主要なプロジェクトディレクトリも検索（package.json, go.mod, Cargo.toml 等がある）
   find /workspace -maxdepth 2 -type f \( -name "package.json" -o -name "go.mod" -o -name "Cargo.toml" -o -name "pyproject.toml" -o -name "Makefile" \) 2>/dev/null | xargs -I {} dirname {}
   ```

2. `$ARGUMENTS` が空の場合、AskUserQuestion ツールを使って、見つかったプロジェクトの中からレビュー対象を選択してもらう

3. 選択されたプロジェクトディレクトリに `.git` があるか確認し、**Git モード** か **非 Git モード** かを決定:
   ```bash
   test -d <project-path>/.git && echo "GIT_MODE" || echo "NON_GIT_MODE"
   ```

### Step 2: レビュー対象の決定

#### Git モードの場合:

1. 現在のブランチと利用可能なブランチを確認:
   ```bash
   git -C <project-path> branch -a
   git -C <project-path> branch --show-current
   ```

2. AskUserQuestion ツールを使って以下を確認:
   - **ベースブランチ**: 比較元となるブランチ（例: main, master, develop）
   - **レビュー対象ブランチ**: レビューしたいブランチ（現在のブランチがデフォルト）

#### 非 Git モードの場合:

1. プロジェクト内のファイル構造を確認:
   ```bash
   find <project-path> -type f -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.rs" 2>/dev/null | head -50
   ```

2. AskUserQuestion ツールを使って以下を確認:
   - **レビュー対象**: レビューしたいファイルまたはディレクトリのパス（複数可）
   - 例: `src/`, `internal/mcp/`, `main.go` など

### Step 3: 変更の概要入力

AskUserQuestion ツールを使って以下を入力してもらう:
- **変更の概要**: この変更の目的や背景を簡潔に説明してもらう
  - 例: 「ユーザー認証機能の追加」「パフォーマンス改善」「バグ修正」
  - 非 Git モードの場合: 「新規実装のレビュー」「コード品質チェック」など

### Step 4: レビュー対象の取得と分析

#### Git モードの場合:

1. ベースブランチとレビュー対象ブランチの差分を取得:
   ```bash
   git -C <project-path> diff <base-branch>...<target-branch> --name-only
   git -C <project-path> diff <base-branch>...<target-branch>
   git -C <project-path> log <base-branch>...<target-branch> --oneline
   ```

2. 変更されたファイルの一覧を記録

#### 非 Git モードの場合:

1. 指定されたファイル/ディレクトリ内のソースコードを収集:
   ```bash
   find <target-path> -type f \( -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.rs" -o -name "*.java" \) 2>/dev/null
   ```

2. 各ファイルの内容を読み取り、レビュー対象として記録

#### 共通:

3. 関連する CLAUDE.md ファイルを収集:
   - プロジェクトルートの CLAUDE.md
   - レビュー対象ファイルのディレクトリにある CLAUDE.md

### Step 5: 並列レビューの実行

**Git モードの場合**: 5つの並列 Sonnet エージェントを起動
**非 Git モードの場合**: 3つの並列 Sonnet エージェント（Git 履歴関連をスキップ）

各エージェントには以下を渡す:
- レビュー対象のファイル内容（Git モード: 差分、非 Git モード: ファイル全体）
- 変更の概要（Step 3で入力されたもの）
- 関連する CLAUDE.md の内容

**エージェント #1: CLAUDE.md コンプライアンスチェック**
- コードが CLAUDE.md のガイドラインに従っているか確認
- 違反があれば具体的な箇所と CLAUDE.md の該当ルールを報告

**エージェント #2: バグスキャン**
- レビュー対象のコードに明らかなバグがないか探す
- 大きなバグに集中し、細かい指摘は避ける

**エージェント #3: 履歴コンテキスト分析**（Git モードのみ）
- 変更されたファイルの git blame と履歴を確認:
  ```bash
  git -C <project-path> log -p --follow -- <file>
  git -C <project-path> blame <file>
  ```
- 過去の変更と矛盾がないか確認
- 以前修正されたバグを再導入していないか確認

**エージェント #4: 過去のコミット分析**（Git モードのみ）
- 同じファイルに対する過去のコミットを確認
- 過去のコミットメッセージから関連する注意点を抽出

**エージェント #5: コードコメントチェック**
- レビュー対象ファイル内のコメントを確認
- コードがコメント内のガイダンスに従っているか確認
- TODO や FIXME コメントへの対応を確認

各エージェントは以下の形式で問題を報告:
```
- ファイル: <file-path>
- 行: <line-number>
- 問題: <description>
- 根拠: <reason> (CLAUDE.md違反 / バグ / 履歴コンテキスト / コメント違反)
```

### Step 6: 信頼度スコアリング

Step 5 で見つかった各問題に対して、Haiku エージェントを起動してスコアリング:

スコアの基準（この基準をエージェントにそのまま渡す）:
- **0**: 確信なし。軽い精査で崩れる誤検知、または既存の問題
- **25**: やや確信あり。実際の問題かもしれないが、誤検知の可能性もある。スタイルの問題の場合、CLAUDE.md で明示されていない
- **50**: まあまあ確信あり。実際の問題だが、些細または実際には発生しにくい。PR全体の中では重要度が低い
- **75**: かなり確信あり。問題を再確認し、実際に発生する可能性が高いことを確認。既存のアプローチでは不十分。機能に直接影響するか、CLAUDE.md で明示されている問題
- **100**: 絶対確信あり。問題を再確認し、確実に発生することを確認。頻繁に発生する

### Step 7: フィルタリングとレポート作成

1. スコアが 80 未満の問題をフィルタリング

2. 最終レポートを以下の形式で出力:

---

## コードレビュー結果

**プロジェクト**: <project-path>
**モード**: Git モード / 非 Git モード
**レビュー対象**:
  - Git モード: <base-branch>...<target-branch>
  - 非 Git モード: <target-files-or-directories>
**変更の概要**: <user-provided-summary>

### 発見された問題

問題が見つかった場合:

**問題 1**: <問題の簡潔な説明>
- ファイル: `<file-path>`
- 行: L<start>-L<end>
- 根拠: <CLAUDE.md / バグ / 履歴コンテキスト / コメント>
- 信頼度: <score>/100

```diff
<該当箇所のコード>
```

---

問題が見つからなかった場合:

### コードレビュー結果

問題は見つかりませんでした。バグと CLAUDE.md コンプライアンスをチェックしました。

---

## 誤検知の例（Step 5 と 6 で考慮すべき）

以下は誤検知として除外すべき:

- バグのように見えるが実際にはバグではないもの
- シニアエンジニアが指摘しないような細かい指摘
- リンター、型チェッカー、コンパイラが検出する問題
- CLAUDE.md で明示的に要求されていない一般的なコード品質の問題
- lint ignore コメントで明示的に無効化されている問題
- 意図的または広範な変更に直接関連する機能変更

Git モードの場合のみ:
- 既存の問題（この PR で導入されたものではない）
- ユーザーが PR で変更していない行の問題

## 注意事項

- ビルドや型チェックは実行しない（それらは別途 CI で実行される）
- gh コマンドは使用しない（ローカルレビューのため）
- 各問題は必ずファイルとラインへのリンクを含める
- TodoWrite ツールを使って進捗を追跡する
