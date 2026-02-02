# ハンズオン

セキュリティ機能を実際に体験するための演習です。

[← README に戻る](../README.ja.md)

---

## デモ環境での git status について

このテンプレートでは、秘匿ファイルの隠蔽を体験できるように、デモ用の秘匿ファイル（`.env`、`secrets/` 内のファイル）を `git add -f` で強制追跡しています。そのため、AI Sandbox 内から git status を見ると隠蔽されたファイルが「削除された」ように見えます。

通常、自分のプロジェクトに適用する場合は秘匿ファイルを `.gitignore` に入れるため、この問題は発生しません。

デモ環境で git status の表示を抑制するには `skip-worktree` を使用します：

```bash
# 既に skip-worktree が設定済みか確認
git ls-files -v | grep ^S

# 隠蔽されたファイルを git status から除外
git update-index --skip-worktree <file>

# 設定を解除する場合
git update-index --no-skip-worktree <file>
```

---

## 秘匿情報隠蔽を体験してみよう

このプロジェクトでは **2つの隠蔽メカニズム** を使い分けています：

| 方法 | 効果 | 用途 |
|------|------|------|
| Docker マウント | ファイル自体が見えない | `.env`、証明書など |
| `.claude/settings.json` | Claude Code がアクセス拒否 | ソースコード内の秘匿情報 |

---

### 方法1: Docker マウントによる隠蔽

秘匿設定の **正常な状態** と **設定漏れの状態** の両方を体験します。

#### ステップ1: 正常な状態を確認

まず、現在の設定で秘匿ファイルが正しく隠蔽されていることを確認します。

```bash
# AI Sandbox 内で実行
# iOS アプリの Config ディレクトリを確認（空に見える）
ls -la demo-apps-ios/SecureNote/Config/

# Firebase 設定ファイルを確認（空または存在しない）
cat demo-apps-ios/SecureNote/GoogleService-Info.plist
```

ディレクトリが空、またはファイルの内容が空であれば、正しく隠蔽されています。

#### ステップ2: 設定漏れを体験する

意図的に設定をコメントアウトして、設定漏れの状態を体験します。

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

> 📝 **まとめ:** Docker マウントによる秘匿設定は、両方の AI Sandbox 環境（DevContainer と CLI Sandbox）で同期する必要があります。設定漏れがあると起動時に検出され、警告が表示されます。

---

### 方法2: .claude/settings.json による制限（保険 + Dockerマウントによる秘匿対象の提案）

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
- **タイミング**: AI Sandbox 起動時に自動実行

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

# マージ結果を確認（AI Sandbox 起動時に作成された）
cat /workspace/.claude/settings.json
```

> 📝 マージは `.sandbox/scripts/merge-claude-settings.sh` で行われます。
