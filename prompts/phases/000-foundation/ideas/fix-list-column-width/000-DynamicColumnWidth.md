# devctl list コマンドのカラム幅動的調整

## 背景 (Background)

現在の `devctl list` コマンドの表示は、各カラムの幅がハードコードされている（`%-24s`, `%-20s` 等）。
そのため、ブランチ名が長い場合にカラムが崩れたり、短い名前のカラムに過大なスペースが割り当てられたりする。

### 現行の表示例（問題のある状態）

```
$ ./bin/devctl list
BRANCH                   FEATURE              CONTAINER            CODE
feat-devctl-scafford     devctl               active               (unknown)
feat-pr-time             -                    (no state)           (unknown)
fix-nested-worktree-deletion -                    (no state)           (unknown)
main                     -                    (no state)           (unknown)
```

- `BRANCH` カラム幅は24文字固定だが、`fix-nested-worktree-deletion` のような長い名前がはみ出す
- `FEATURE` や `CONTAINER` は実際の内容に比べて過大な固定幅が割り当てられている

### 現行の実装

- `listing.go` の `FormatTable()` 関数がフォーマット文字列 `%-24s`, `%-20s`, `%-16s` を使って固定幅で出力
- カラム幅の計算ロジックは一切存在しない

## 要件 (Requirements)

### 必須要件

1. **カラム幅の動的計算**: 各カラムの幅を、全行のセル文字列長の最大値から動的に算出する
   - ヘッダー文字列の長さもカラム幅の計算対象に含める
   - 最終カラム（PATH なしなら CODE、PATH ありなら PATH）にはパディングを付けない

2. **カラム間パディング**: 隣接するカラム間のスペースを設定可能にする
   - デフォルト: 2スペース
   - 環境変数 `DEVCTL_LIST_PADDING` で上書き可能

3. **最大カラム幅の制限**: セル文字列が長すぎる場合にトランケーションする
   - デフォルト上限: 40文字
   - 環境変数 `DEVCTL_LIST_WIDTH` で上書き可能
   - トランケーション方法: 上限値から4文字を引いた位置でカットし、末尾に ` ...` を付与する
     - 例: 上限40文字 → 36文字でカットし ` ...` を付加（合計40文字）
   - ヘッダー文字列はトランケーション対象外とする（ヘッダー自体は短いため）

4. **`--full` オプション**: トランケーションを無効化するフラグ
   - `--full` 指定時はカラム幅上限を適用せず、全文を表示する
   - カラム幅は全行のセル文字列の最大値がそのまま使われる

5. **`--env` オプション**: レポートの Environment Variables セクション全体の表示を制御するフラグ
   - `--env` 指定時のみ、レポートに `## Environment Variables` セクションを表示する
   - このセクションには既存の `DEVCTL_*` 環境変数に加え、`DEVCTL_LIST_WIDTH` と `DEVCTL_LIST_PADDING` も含める
   - 表示形式（既存のレポートと同一フォーマット）:
     ```
     ## Environment Variables
     | Variable | Value |
     |---|---|
     | DEVCTL_EDITOR | *(not set, default: cursor)* |
     | DEVCTL_CMD_CODE | *(not set, default: code)* |
     | ... | ... |
     | DEVCTL_LIST_WIDTH | *(not set, default: 40)* |
     | DEVCTL_LIST_PADDING | *(not set, default: 2)* |
     ```
   - `--env` 未指定時は Environment Variables セクション丸ごと非表示（レポートに `EnvVars` を渡さない）

6. **環境変数の優先順位**:
   - `DEVCTL_LIST_WIDTH`: 設定されていればその値を使用（デフォルト: 40）
   - `DEVCTL_LIST_PADDING`: 設定されていればその値を使用（デフォルト: 2）
   - `--full` フラグは `DEVCTL_LIST_WIDTH` の設定を無視する（トランケーション自体を無効化）

### 任意要件

- なし

## 実現方針 (Implementation Approach)

### 変更対象ファイル

#### 1. `features/devctl/internal/listing/listing.go`

- `FormatTable()` のシグネチャを変更し、テーブル表示オプションを受け取れるようにする
  - `TableOptions` 構造体を新設:
    ```go
    type TableOptions struct {
        ShowPath  bool
        Full      bool  // --full: disable truncation
        MaxWidth  int   // DEVCTL_LIST_WIDTH (default: 40)
        Padding   int   // DEVCTL_LIST_PADDING (default: 2)
    }
    ```
- カラム幅計算ロジックを追加:
  1. 全行のセル文字列長を走査し、各カラムの最大幅を算出
  2. `Full` が false の場合、最大幅を `MaxWidth` でキャップ
  3. 各カラムに `Padding` を加えたフォーマット文字列を動的に生成
- トランケーション関数を追加:
  - `truncateCell(s string, maxWidth int) string`: 文字列が `maxWidth` を超える場合に `maxWidth - 4` 文字でカットし ` ...` を付加

#### 2. `features/devctl/cmd/list.go`

- `--full` フラグを追加:
  ```go
  var flagListFull bool
  listCmd.Flags().BoolVar(&flagListFull, "full", false, "Disable column truncation")
  ```
- `--env` フラグを追加:
  ```go
  var flagListEnv bool
  listCmd.Flags().BoolVar(&flagListEnv, "env", false, "Show environment variables section")
  ```
- `runListBranches()` 内で `DEVCTL_LIST_WIDTH` と `DEVCTL_LIST_PADDING` 環境変数を読み取り、`TableOptions` を構築
- `--env` 指定時のみ `CollectEnvVars()` を呼び出してレポートの `EnvVars` に設定する
- `--env` 未指定時は `EnvVars` を空にしてレポートに渡す（セクション非表示）
- `listing.FormatTable()` の呼び出しを `TableOptions` を渡す形に変更

#### 3. `features/devctl/cmd/common.go`

- `knownEnvVars` に `DEVCTL_LIST_WIDTH`（デフォルト: `"40"`）と `DEVCTL_LIST_PADDING`（デフォルト: `"2"`）を追加

#### 4. `features/devctl/internal/listing/listing_test.go`

- `TestFormatTable` テストケースを更新:
  - 新しいシグネチャ（`TableOptions`）に対応
  - 動的カラム幅が正しく計算されていることを検証
- トランケーションのテストケースを追加:
  - 長いブランチ名がトランケーションされること
  - `Full: true` でトランケーションされないこと
  - カスタム `MaxWidth` / `Padding` が適用されること

### 出力イメージ

**変更後（通常）:**

```
BRANCH                            FEATURE  CONTAINER       CODE
feat-devctl-scafford              devctl   active          (unknown)
feat-pr-time                      -        (no state)      (unknown)
fix-nested-worktree-deletion      -        (no state)      (unknown)
main                              -        (no state)      (unknown)
```

**長い名前がある場合（トランケーション動作）:**

```
BRANCH                                FEATURE  CONTAINER       CODE
feat-devctl-scafford                  devctl   active          (unknown)
this-is-a-very-long-branch-name ...   -        (no state)      (unknown)
main                                  -        (no state)      (unknown)
```

**`--env` 指定時（レポート出力に含まれる）:**

```
## Environment Variables
| Variable | Value |
|---|---|
| DEVCTL_EDITOR | *(not set, default: cursor)* |
| DEVCTL_CMD_CODE | *(not set, default: code)* |
| ... | ... |
| DEVCTL_LIST_WIDTH | *(not set, default: 40)* |
| DEVCTL_LIST_PADDING | *(not set, default: 2)* |
```

## 検証シナリオ (Verification Scenarios)

### シナリオ 1: 動的カラム幅

1. 長さの異なるブランチ名（短い名前・長い名前）を持つ状態で `devctl list` を実行する
2. 各カラムの幅がセル内容の最長文字列に合わせて動的に調整される
3. カラム間のスペースが2文字分（デフォルト）になっている

### シナリオ 2: トランケーション

1. 40文字を超えるブランチ名を持つ状態で `devctl list` を実行する
2. そのブランチ名が36文字でカットされ、末尾に ` ...` が付加される
3. 合計表示幅が40文字になっている

### シナリオ 3: `--full` オプション

1. 40文字を超えるブランチ名を持つ状態で `devctl list --full` を実行する
2. ブランチ名がトランケーションされずに全文表示される

### シナリオ 4: 環境変数による設定変更

1. `DEVCTL_LIST_WIDTH=30 DEVCTL_LIST_PADDING=4 devctl list` を実行する
2. トランケーション上限が30文字、カラム間パディングが4スペースになる

### シナリオ 5: `--env` オプション

1. `devctl list --env` を実行する
2. レポートに `## Environment Variables` セクションが表示される
3. `DEVCTL_LIST_WIDTH` と `DEVCTL_LIST_PADDING` がテーブルに含まれる
4. `devctl list`（`--env` なし）では Environment Variables セクションが丸ごと表示されない

### シナリオ 6: `--full` と環境変数の組み合わせ

1. `DEVCTL_LIST_WIDTH=20 devctl list --full` を実行する
2. `--full` が優先され、トランケーションは行われない

## テスト項目 (Testing for the Requirements)

### 単体テスト

| 要件 | テスト内容 | テストファイル |
|------|-----------|--------------|
| R1 | カラム幅がセル内容の最大長に基づいて動的に計算されること | `listing_test.go` の `TestFormatTable` を更新 |
| R2 | カラム間パディングが `Padding` に従うこと | `listing_test.go` に新規テスト追加 |
| R3 | `MaxWidth` を超えるセルがトランケーションされること | `listing_test.go` に `TestTruncateCell` を追加 |
| R3 | トランケーション後の文字列が `maxWidth - 4` 文字 + ` ...` であること | 同上 |
| R4 | `Full: true` の場合トランケーションが無効化されること | `listing_test.go` にテスト追加 |
| R6 | `MaxWidth` と `Padding` のカスタム値が正しく適用されること | `listing_test.go` にテスト追加 |

### ビルド検証

```bash
# 全体ビルド & 単体テスト
./scripts/process/build.sh
```

### 統合テスト

```bash
# 統合テスト実行
./scripts/process/integration_test.sh
```
