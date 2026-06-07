# 予約ブランチ名の保護

## 背景 (Background)

`devctl close` コマンドは、指定されたブランチのworktreeを削除し、ブランチ自体も削除する破壊的な操作を行う。`main` や `master` といったデフォルトブランチに対して誤ってこのコマンドを実行すると、リポジトリの主要ブランチが消失し、復旧に手間がかかる深刻な問題が発生する。

現状、ブランチ名に対するバリデーションは行われておらず、任意のブランチ名がすべてのコマンドで受け入れられる状態である。

## 要件 (Requirements)

### 必須要件

1. **予約ブランチ名の定義**: `main` および `master` を予約ブランチ名（Reserved Branch Names）として定義する。
2. **全コマンドでの一律拒否**: ブランチ名を第1引数として受け取る全サブコマンドにおいて、予約ブランチ名が指定された場合はエラーを返し、処理を実行しない。
   - 対象コマンド（全8コマンド）:
     - `up <branch> [feature]`
     - `close <branch> [feature]`
     - `down <branch> <feature>`
     - `open <branch> [feature]`
     - `shell <branch> <feature>`
     - `exec <branch> <feature> -- <command...>`
     - `status <branch> [feature]`
     - `pr <branch> [feature]`
3. **エラーメッセージ**: 予約ブランチ名が指定された場合、わかりやすいエラーメッセージを返す。
   - 例: `"main" is a reserved branch name and cannot be used with devctl commands`
4. **大文字・小文字の区別**: 完全一致のみを対象とする（`Main`, `MAIN` は許容される）。

### 任意要件

5. **将来的な拡張性**: 予約ブランチ名リストをスライス定数として定義し、将来的に `develop` や `release` などの追加が容易な設計にする。

## 実現方針 (Implementation Approach)

### 変更箇所

全コマンドがブランチ名の解析に `InitContext()` 関数（`features/devctl/cmd/common.go`）を使用している。ここにバリデーションを追加することで、全コマンドに一律に保護を適用できる。

```
InitContext(args)
  └─ ParseBranchFeature(args)  → branch を取得
  └─ validateBranchName(branch) を追加  ← ★ ここで検証
```

### 主要コンポーネント

1. **`features/devctl/cmd/common.go`**:
   - 予約ブランチ名リスト `reservedBranchNames` をパッケージレベルで定義
   - `validateBranchName(branch string) error` 関数を追加
   - `InitContext()` 内で `ParseBranchFeature` の直後に呼び出す

2. **`features/devctl/cmd/exec_cmd.go`**:
   - `exec` コマンドは `InitContext` を直接呼び出す前に独自でブランチ名をパースしている
   - `InitContext` に渡す前にバリデーションが通るため、追加変更は不要
   - ただし、`exec` コマンドの `beforeDash` からブランチ名を取得する箇所も `InitContext` 経由でバリデーションされることを確認

### 設計ポイント

- **一元管理**: バリデーションロジックを `InitContext` に集約し、コマンドごとの個別対応を不要にする
- **単純さ**: 予約ブランチ名は文字列スライスで管理し、過度な抽象化を避ける
- **テスト容易性**: バリデーション関数を独立させ、ユニットテストを書きやすくする

## 検証シナリオ (Verification Scenarios)

### シナリオ1: `close` コマンドで `main` を指定

1. `devctl close main` を実行する
2. エラーメッセージ `"main" is a reserved branch name and cannot be used with devctl commands` が表示される
3. worktree の削除やブランチ削除は一切行われない

### シナリオ2: `close` コマンドで `master` を指定

1. `devctl close master` を実行する
2. 同様のエラーメッセージが表示される
3. 処理は行われない

### シナリオ3: `up` コマンドで `main` を指定

1. `devctl up main` を実行する
2. エラーメッセージが表示される
3. worktree の作成は行われない

### シナリオ4: 他のコマンドでも同様に拒否される

1. `devctl down main devctl`, `devctl open main`, `devctl shell main devctl`, `devctl exec main devctl -- ls`, `devctl status main`, `devctl pr main` をそれぞれ実行する
2. すべてエラーメッセージが表示される

### シナリオ5: 通常のブランチ名は影響を受けない

1. `devctl up my-feature` を実行する
2. 通常通り処理が実行される（予約ブランチ名チェックを通過する）

### シナリオ6: 大文字小文字を区別する

1. `devctl up Main` を実行する
2. 通常通り処理が実行される（`Main` は予約名 `main` と完全一致しないため許容）

## テスト項目 (Testing for the Requirements)

### 自動テスト

#### ユニットテスト: `features/devctl/cmd/common_test.go`

| テストケース | 検証内容 |
|---|---|
| `TestValidateBranchName_Main` | `main` を渡すとエラーが返ること |
| `TestValidateBranchName_Master` | `master` を渡すとエラーが返ること |
| `TestValidateBranchName_Normal` | `my-feature` など通常名ではエラーが返らないこと |
| `TestValidateBranchName_CaseSensitive` | `Main`, `MASTER` ではエラーが返らないこと |
| `TestInitContext_ReservedBranch` | `InitContext([]string{"main"})` がエラーを返すこと |

#### 検証コマンド

```bash
# 全体ビルド & 単体テスト
./scripts/process/build.sh
```
