# 016: ApplyPostActions のテンプレート変数未展開バグの修正

## 背景 (Background)

`tt scaffold feature` コマンドで、`base_dir` にテンプレート変数（例: `features/{{feature_name}}`）を含むscaffoldを適用した際、post-actions（ファイルパーミッション設定）の実行時に以下のエラーが発生する:

```
scaffold apply failed: failed to apply post-actions for feature/kotoshiro-go-mcp:
GetFileAttributesEx c:\Users\yamya\...\features\{{feature_name}}: The system cannot find the file specified.
```

### 原因分析

`ApplyFiles` 関数内では `placement.BaseDir` に含まれるテンプレート変数 `{{feature_name}}` を `ProcessTemplatePath` で展開してからファイルを配置している（[applier.go:54-60](file:///c:/Users/yamya/myprog/tokotachi/pkg/scaffold/applier.go#L54-L60)）。

しかし、`ApplyPostActions` の呼び出し箇所では、`placement.BaseDir` をテンプレート変数を展開せずにそのまま渡しているため、`applyFilePermissions` 内の `filepath.WalkDir` が `features/{{feature_name}}` という存在しないディレクトリをウォークしようとしてエラーとなる。

影響箇所は以下の2つ:

1. **`applySingleScaffold`** ([scaffold.go:276](file:///c:/Users/yamya/myprog/tokotachi/pkg/scaffold/scaffold.go#L276)):
   ```go
   ApplyPostActions(placement.PostActions, opts.RepoRoot, placement.BaseDir)
   ```
2. **`applyDependencyChain`** ([scaffold.go:339](file:///c:/Users/yamya/myprog/tokotachi/pkg/scaffold/scaffold.go#L339)):
   ```go
   ApplyPostActions(placement.PostActions, opts.RepoRoot, placement.BaseDir)
   ```

どちらの箇所でも、`ApplyFiles` のように `optionValues` を使った `ProcessTemplatePath` 展開が行われていない。

## 要件 (Requirements)

### 必須要件

1. **`applySingleScaffold` の修正**: `ApplyPostActions` 呼び出し前に `placement.BaseDir` をテンプレート変数展開する
2. **`applyDependencyChain` の修正**: 同様に `placement.BaseDir` をテンプレート変数展開する
3. **テンプレート変数展開ロジック**: 既存の `ProcessTemplatePath` 関数を再利用し、`optionValues`（`--v` フラグで提供された値）を使用すること
4. **動作仕様**: `ApplyFiles` と同じ展開結果が `ApplyPostActions` の `baseDir` にも適用されること
5. **テスト追加**: `ApplyPostActions` にテンプレート変数を含む `baseDir` を渡した場合の単体テストを追加する

### 任意要件

- `ApplyPostActions` 関数自体の内部でテンプレート変数を展開するように改修することも検討可能だが、呼び出し側で展開して渡す方がシンプルである（`ApplyFiles` と同じパターン）

## 実現方針 (Implementation Approach)

### 修正方針

修正は2つのアプローチが考えられるが、**アプローチ A（呼び出し側で展開）** を推奨する。

#### アプローチ A: 呼び出し側で展開（推奨）

`applySingleScaffold` と `applyDependencyChain` の `ApplyPostActions` 呼び出し前に、`baseDir` のテンプレート変数展開を行う:

```go
// applySingleScaffold 内（scaffold.go:276付近）
baseDir := placement.BaseDir
if optionValues != nil && strings.Contains(baseDir, "{{") {
    var err error
    baseDir, err = ProcessTemplatePath(baseDir, optionValues)
    if err != nil {
        return fmt.Errorf("failed to process base_dir template for post-actions: %w", err)
    }
}
if err := ApplyPostActions(placement.PostActions, opts.RepoRoot, baseDir); err != nil {
    return fmt.Errorf("failed to apply post-actions: %w", err)
}
```

**利点**: `ApplyFiles` と同じパターンであり、`ApplyPostActions` のインターフェースを変更せずに済む。

#### アプローチ B: `ApplyPostActions` 内部で展開

`ApplyPostActions` の引数に `optionValues` を追加し、関数内部で展開する。

**欠点**: 関数シグネチャの変更が必要で、既存のテストや呼び出し箇所すべてに影響する。

### 修正対象ファイル

| ファイル | 修正内容 |
|---|---|
| `pkg/scaffold/scaffold.go` | `applySingleScaffold` と `applyDependencyChain` の `ApplyPostActions` 呼び出し前に `baseDir` のテンプレート展開を追加 |
| `pkg/scaffold/applier_test.go` | テンプレート変数を含む `baseDir` でのpost-actionsテストを追加 |

## 検証シナリオ (Verification Scenarios)

1. `base_dir` に `features/{{feature_name}}` を使用するscaffold（例: `feature/kotoshiro-go-mcp`）を `--v feature_name=test` オプション付きで実行する
2. `ApplyFiles` によるファイル配置が `features/test/` ディレクトリに正常に行われる
3. `ApplyPostActions` のファイルパーミッション設定が `features/test/` ディレクトリを正しくウォークして適用される
4. エラーなしでscaffold applyが完了する

## テスト項目 (Testing for the Requirements)

### 単体テスト

以下のテストを `pkg/scaffold/applier_test.go` に追加する:

- **`TestApplyPostActions_WithTemplateBaseDir`**: `baseDir` にテンプレート変数 `{{feature_name}}` を含む値を渡した場合、展開後のパスでpost-actionsが正しく実行されることを検証する

> [!IMPORTANT]
> ただし、本バグの修正は `ApplyPostActions` の**外側**（呼び出し元）で `baseDir` を展開するアプローチのため、`ApplyPostActions` 自体のテストでは「展開済みパス」が渡されることをテストする形になる。
> 呼び出し元の結合部分の検証には統合テストが必要。

### 自動テスト実行コマンド

```bash
# 全体ビルド & 単体テスト
scripts/process/build.sh

# 統合テスト（scaffold関連）
scripts/process/integration_test.sh
```

### テスト詳細

| 要件 | テスト方法 | 検証コマンド |
|---|---|---|
| `applySingleScaffold` での展開 | 統合テスト: `feature/axsh-go-standard` 等の `base_dir` にテンプレート変数を含むscaffoldのapply | `scripts/process/integration_test.sh` |
| `applyDependencyChain` での展開 | 統合テスト: 依存チェーンを持ちかつ `base_dir` にテンプレート変数を含むscaffoldのapply | `scripts/process/integration_test.sh` |
| テンプレート変数なしの場合の非影響 | 既存単体テスト: `TestApplyPostActions_WithFilePermissions` 等 | `scripts/process/build.sh` |
