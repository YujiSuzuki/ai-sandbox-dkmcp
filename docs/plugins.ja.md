# マルチプロジェクトワークスペースでのプラグイン使用

[← README.ja.md に戻る](../README.ja.md)

**注意**: このドキュメントは Claude Code 専用です。Gemini Code Assist では使えません。

**注意**: それぞれのプロジェクトが独立したGitで管理されている場合の説明です。
モノレポの場合は普通のプラグインの使い方で問題ありません。マルチリポ（独立したリポジトリ）の場合は工夫が必要でその説明になります。

## 前提

ワークスペースに複数のプロジェクトがあったとします：

```
workspace/
├── your-apps/           # API + React Web (Node.js)
├── your-apps-ios/       # iOS App (Swift)
└── your-other-projects/ # other
```

## マルチリポ（独立したリポジトリ）の構成の場合

各プロジェクトが **独立した Git リポジトリ** を持つため、PR やブランチも各プロジェクトに独立して存在します：

```
workspace/
├── your-apps/           # 独立の Git リポジトリ
│   ├── .git/
│   ├── main, develop ブランチ
│   ├── PR（your-apps 専用）
│   └── コミット履歴
│
├── your-apps-ios/       # 独立の Git リポジトリ
│   ├── .git/
│   ├── main, develop ブランチ
│   ├── PR（your-apps-ios 専用）
│   └── コミット履歴
│
└── your-other-projects/       # 独立の Git リポジトリ
    ├── .git/
    ├── main, develop ブランチ
    ├── PR（your-other-projects 専用）
    └── コミット履歴

```

## 重要な前提：プラグインは各プロジェクトディレクトリ内でのみ動作

> [!IMPORTANT]
> [Claude Code のプラグイン](https://github.com/anthropics/claude-code/blob/main/plugins/README.md)（`/code-review` 等）は、**プラグインを実行するプロジェクトディレクトリ内でのみ動作** します：
>
> ```
> /workspace/          ← ここで /code-review しても意味がない
>   ├── your-apps/     ← cd してから /code-review 実行
>   ├── your-apps-ios/ ← cd してから /code-review 実行
>   └── your-other-project/       ← cd してから /code-review 実行
> ```


**つまり：**
- `your-apps` のコードレビューは、`your-apps` ディレクトリで `/code-review` 実行 → `your-apps` の PR/ブランチをレビュー
- `your-apps-ios` のコードレビューは、`your-apps-ios` ディレクトリで `/code-review` 実行 → `your-apps-ios` の PR/ブランチをレビュー
- 各プロジェクトのプラグインは、そのプロジェクトのコードのみを対象

## モノレポ vs マルチリポ

**モノレポ構成の場合:**
```bash
# workspace ルートで /code-review 実行可能
/workspace で /code-review → すべてのプロジェクトのコードをレビュー対象にできる
```

**マルチリポの場合:**
```bash
# 各プロジェクトディレクトリで /code-review 実行が必要
cd /workspace/your-apps && # code-review → your-apps のコードのみをレビュー
cd /workspace/your-apps-ios && # code-review → your-apps-ios のコードのみをレビュー
```

## マルチリポ（独立したリポジトリ）の場合の工夫

Claude Code のプラグイン /code-review を例に説明します。


1. **通常通りプラグインをインストール**

インストール方法は [公式ページ](https://github.com/anthropics/claude-code/blob/main/plugins/README.md)を参照してください

> [!TIP]
>    ```bash
>    claude --help
>    claude plugin --help
>    claude plugin install --help
>    ```

例：

```
node@671e8b3485a2:/workspace$ claude plugin install code-review
Installing plugin "code-review"...
✔ Successfully installed plugin: code-review@claude-plugins-official (scope: user)
node@671e8b3485a2:/workspace$

```

インストール後の Claude Code の 確認

```
❯ /plugin
─────────────────────────────────────────────────────────────────────
 Plugins  Discover   Installed   Marketplaces  (←/→ or tab to cycle)

   Local
 ❯ dkmcp MCP · ✔ connected

   User
   code-review Plugin · claude-plugins-official · ✔ enabled
   gopls-lsp Plugin · claude-plugins-official · ✔ enabled

  Space to toggle · Enter to details · escape to back
```

2. ** インストールしたプラグインのラッパーコマンドの作成を依頼**

code-review は通常 gh (GitHub CLI) コマンドが使える前提で動作しますので、ここでは、gh に 依存しないローカル環境でのみ処理するようにしてみます。（お好みで調整してください）

   Claude Code に以下のように依頼：

   ```
   code-review プラグインを解析して、親ディレクトリから使える
   カスタムコマンドを作成してください。
   要件：
   - workspaceの配下でGit管理されているプロジェクトのどれをレビューするか選択できる
   - レビュー対象のブランチをユーザーに確認する
   - PR の概要や目的を入力してもらう
   - 選択したプロジェクトディレクトリで、code-review プラグインと同じレビューを実行
   - github にはアクセスしないで、gh を使用しない方法で処理すること
   ```

   > **すぐ使えるコマンド**: 上記の要件に、さらに以下の改良を加えたコマンドを用意しています：
   > - Git リポジトリがなくても動作（Non-Git モード対応）
   > - 専門レビュー5種類（general / security / performance / architecture / prompt）
   > - バッチスコアリング + Validation の2段階検証で偽陽性を削減
   >
   > インストールスクリプトで簡単に追加できます：
   > ```bash
   > .sandbox/scripts/install-commands.sh --list       # 利用可能なコマンドを確認
   > .sandbox/scripts/install-commands.sh ais-local-review  # ais-local-review をインストール
   > .sandbox/scripts/install-commands.sh --all         # 全コマンドをインストール
   > ```

3. AIのカスタムコマンドの作成作業が完了したら、AIを再起動しカスタムコマンドを認識させます。


```
❯ /exit
  ⎿  Bye!
```
の後にターミナルから

```
$ claude
  or
$ claude  --allow-dangerously-skip-permissions
```


## 上記で作成したカスタムコマンドの活用例

- 下記の例では `install-commands.sh` でインストールした付属コマンドを使用しています。

**シナリオ: iOS アプリのログイン機能がうまくいかない**

1. `/ais-local-review` を実行し、**your-apps-ios** を選択
   → iOS のログイン画面コードをレビュー

2. `/ais-local-review` を実行し、**your-apps** を選択
   → API の認証エンドポイントをレビュー

3. Claude Code に DockMCP でログを確認してもらう
   → API コンテナのエラーログを確認

このワークスペース構成だからこそ、複数プロジェクト間の問題を総合的に調査できます。
