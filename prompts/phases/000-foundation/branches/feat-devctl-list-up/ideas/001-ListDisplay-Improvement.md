# devctl list コマンドの表示改善: カラム構成の見直し

## 背景 (Background)

現在の `devctl list` コマンド（引数なしのブランチ概要モード）の出力は以下のようになっている:

```
BRANCH                   FEATURES
feat-catalog             devctl[active]
feat-devctl-list-up      (no state)
feat-devctl-scafford     devctl[active]
main                     (no state)
```

この表示には以下の問題がある:

1. **`FEATURES` カラムに情報が混在している**: feature 名（例: `devctl`）と状態（例: `active`）が `devctl[active]` のように1つのカラムに詰め込まれており、視認性が悪い
2. **`(no state)` の表示位置が不適切**: state ファイルが存在しないことを示す `(no state)` が `FEATURES` カラムに表示されているが、これは feature 情報ではなく**状態**に関する情報である
3. **カラム名が複数形**: `FEATURES` は複数形だが、実際には1つの feature のみ表示されるケースが多い

### 現行の実装

- `listing.go` の `featuresLabel()` 関数が `FEATURES` カラムの表示文字列を生成
- feature 名とステータスを `devctl[active]` の形式で結合している
- state ファイルがない場合は `(no state)` を返す

## 要件 (Requirements)

### 必須要件

1. **`FEATURES` カラムを `FEATURE` に名称変更する**
   - ヘッダー行の `FEATURES` を `FEATURE` に変更
   - 既存の JSON 出力フォーマットは変更しない（後方互換性維持）

2. **`STATE` カラムを新設する**
   - `FEATURE` カラムと `PATH` カラムの間に `STATE` カラムを追加
   - 各ブランチの状態をこのカラムに表示する
   - テーブルヘッダーの表示順: `BRANCH`, `FEATURE`, `STATE`, (`PATH`)

3. **表示内容の配置変更**
   - feature 名はそのまま `FEATURE` カラムに表示（例: `devctl`）
   - feature のステータス（`active` 等）は `STATE` カラムに表示
   - state ファイルがない場合: `FEATURE` = `-`, `STATE` = `(no state)`
   - main worktree の場合: `FEATURE` = `-`, `STATE` = `(main worktree)`
   - 複数 feature がある場合: feature 名をカンマ区切り、state は代表的なものを表示

4. **`--path` フラグの動作維持**
   - `--path` フラグ付きの場合は、`STATE` カラムの後に `PATH` カラムを表示

5. **JSON 出力は変更しない**
   - `--json` フラグによる JSON 出力のフォーマットは現状維持
   - データ構造（`BranchInfo`, `FeatureInfo`）の変更は不要

### 任意要件

- カラム幅の調整: 各カラムの幅は内容に合わせて適切に設定する

## 実現方針 (Implementation Approach)

### 変更対象ファイル

#### 1. `features/devctl/internal/listing/listing.go`

- `featuresLabel()` 関数を `featureColumn()` と `stateColumn()` の2つに分割
  - `featureColumn(bi BranchInfo) string`: feature 名のみを返す（例: `devctl`, `-`）
  - `stateColumn(bi BranchInfo) string`: 状態のみを返す（例: `active`, `(no state)`, `(main worktree)`）
- `FormatTable()` 関数のヘッダーとボディの出力形式を変更:
  - ヘッダー: `BRANCH`, `FEATURE`, `STATE`, (`PATH`)
  - ボディ: 各カラムに対応する値を出力

#### 2. `features/devctl/internal/listing/listing_test.go`

- `TestFormatTable` テストケースを更新:
  - ヘッダーの `FEATURES` → `FEATURE` の変更を反映
  - `STATE` カラムの存在を検証
  - 各カラムの値が正しい位置に表示されることを検証

### 出力イメージ

**変更前:**

```
BRANCH                   FEATURES
feat-catalog             devctl[active]
feat-devctl-list-up      (no state)
feat-devctl-scafford     devctl[active]
main                     (no state)
```

**変更後（`--path` なし）:**

```
BRANCH                   FEATURE              STATE
feat-catalog             devctl               active
feat-devctl-list-up      -                    (no state)
feat-devctl-scafford     devctl               active
main                     -                    (main worktree)
```

**変更後（`--path` 付き）:**

```
BRANCH                   FEATURE              STATE                PATH
feat-catalog             devctl               active               /path/to/work/feat-catalog
feat-devctl-list-up      -                    (no state)           /path/to/work/feat-devctl-list-up
main                     -                    (main worktree)      /path/to/repo
```

## 検証シナリオ (Verification Scenarios)

### シナリオ 1: カラムヘッダーの確認

1. `devctl list` を引数なしで実行する
2. ヘッダー行に `BRANCH`, `FEATURE`, `STATE` の3カラムが表示される
3. `FEATURES`（複数形）は表示されない

### シナリオ 2: feature がある場合の表示

1. state ファイルに `devctl` feature（status: active）が登録されたブランチがある状態で `devctl list` を実行
2. `FEATURE` カラムに `devctl` が表示される
3. `STATE` カラムに `active` が表示される
4. `devctl[active]` のような結合表示はされない

### シナリオ 3: state ファイルがない場合の表示

1. worktree は存在するが state ファイルがないブランチがある状態で `devctl list` を実行
2. `FEATURE` カラムに `-` が表示される
3. `STATE` カラムに `(no state)` が表示される

### シナリオ 4: main worktree の表示

1. `devctl list` を実行する
2. main worktree の行で `FEATURE` カラムに `-` が表示される
3. `STATE` カラムに `(main worktree)` が表示される

### シナリオ 5: `--path` フラグ付きの表示

1. `devctl list --path` を実行する
2. ヘッダー行に `BRANCH`, `FEATURE`, `STATE`, `PATH` の4カラムが表示される
3. `PATH` カラムに各 worktree のパスが表示される

## テスト項目 (Testing for the Requirements)

### 単体テスト

| 要件 | テスト内容 | テストファイル |
|------|-----------|--------------|
| R1 | ヘッダーに `FEATURE`（単数形）が表示されること | `listing_test.go` の `TestFormatTable` を更新 |
| R1 | ヘッダーに `FEATURES`（複数形）が表示されないこと | `listing_test.go` の `TestFormatTable` を更新 |
| R2 | ヘッダーに `STATE` カラムが表示されること | `listing_test.go` の `TestFormatTable` を更新 |
| R3 | feature 名と state が別カラムに分離されていること | `listing_test.go` の `TestFormatTable` を更新 |
| R3 | state なしの場合に `-` と `(no state)` が表示されること | `listing_test.go` の `TestFormatTable` を更新 |
| R4 | `--path` 付きで `PATH` カラムが表示されること | `listing_test.go` の `TestFormatTable` を更新 |
| R5 | JSON 出力が変更されていないこと | `listing_test.go` の `TestFormatJSON`（変更不要） |

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
