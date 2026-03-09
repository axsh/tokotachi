# Scaffold インタラクティブ入力の改善

## 背景 (Background)

`tt scaffold` コマンドでテンプレートを適用する際、以下の2つの問題が発生している。

### 問題1: オプションの重複問い合わせ

`scaffold.Run()` でオプション値を収集した後、実行確認を経て `scaffold.Apply()` が呼ばれるが、`Apply()` 内部でも再度 `CollectOptionValues()` が呼ばれるため、ユーザーに同じ質問が2回表示される。

```
$ tt scaffold feature axsh-go-standard --cwd
? feature_name (Feature name): sample       ← 1回目 (Run内)
...
Proceed? [y/N]: y
? feature_name (Feature name): sample       ← 2回目 (Apply内)
```

**原因箇所**:
- `scaffold.go` L105: `Run()` 内の `CollectOptionValues()` 呼び出し
- `scaffold.go` L228: `Apply()` 内の `CollectOptionValues()` 呼び出し

### 問題2: `required: false` のオプションが問い合わせされない

現在の `CollectOptionValues()` は、`required: true` の項目のみインタラクティブに問い合わせ、`required: false` の項目はデフォルト値を無条件で適用してしまう。ユーザーは `required` に関わらず、すべてのオプションをインタラクティブに確認・入力したい。

```go
// 現在のロジック (template.go L55-58)
if opt.Default != "" && !opt.Required {
    values[opt.Name] = opt.Default  // ← デフォルトを無条件適用
    continue
}
```

## 要件 (Requirements)

### 必須要件

1. **重複質問の解消**: オプション値の問い合わせは1回のみ行い、`Run()` で収集した値を `Apply()` へ引き継ぐこと
2. **全オプションのインタラクティブ問い合わせ**: `required` の値に関わらず、すべてのオプションをインタラクティブにユーザーへ問い合わせること
3. **デフォルト値の表示と適用**: プロンプト表示時にデフォルト値を表示し、ユーザーが空Enter（入力なし）した場合はデフォルト値を適用すること
4. **`--v key=value` オプションの追加**: 指定されたキーのオプションをインタラクティブ無しで直接値設定できること。複数指定可能とすること（例: `--v feature_name=foo --v go_module=bar`）
5. **`--default` オプションの追加**: 指定した場合、`required: false` の項目はインタラクティブ問い合わせをスキップし、デフォルト値を自動適用すること。`required: true` の項目は引き続き問い合わせること
6. **`--yes` オプションとの組み合わせ**: `--yes` が指定された場合でも、未提供の `required: true` オプションがあればインタラクティブに問い合わせること（現在の動作を維持）

### 任意要件

7. **`gh pr create` 風のプロンプトUI**: `gh pr create` のような視覚的に美しいインタラクティブプロンプトを実装すること
   - `?` マークの色付き表示（グリーン）
   - デフォルト値のヒント表示（`(default-value)`）
   - 入力値の視覚的フィードバック

## 実現方針 (Implementation Approach)

### 1. `RunOptions` への `OptionValues` フィールド追加

`Run()` で収集したオプション値を保持し、`Apply()` へ引き継ぐ仕組みを導入する。

```go
// RunOptions に追加
type RunOptions struct {
    // ... 既存フィールド ...
    OptionOverrides map[string]string // --v で指定された値
    UseDefaults     bool              // --default フラグ
}
```

`Plan` 構造体にも `OptionValues` を保持するフィールドを追加し、`Run()` → `Apply()` の間で値を引き継ぐ。

```go
type Plan struct {
    // ... 既存フィールド ...
    OptionValues map[string]string // 収集済みオプション値
}
```

### 2. `CollectOptionValues` の改修

現在の `CollectOptionValues` を以下のように変更する:

- すべてのオプション（`required` に関わらず）をインタラクティブに問い合わせる
- デフォルト値がある場合はプロンプトに表示する
- 空入力時はデフォルト値を適用する
- `required: true` で空入力かつデフォルトなしの場合はエラーとする
- `UseDefaults` フラグが `true` の場合は `required: false` の項目を自動適用する

```go
// 新しいシグネチャ
func CollectOptionValues(options []Option, provided map[string]string,
    reader io.Reader, writer io.Writer, useDefaults bool) (map[string]string, error)
```

### 3. `Apply()` の改修

`Apply()` 内で `CollectOptionValues()` を呼ばず、`Plan.OptionValues` から値を取得する。

### 4. CLI フラグの追加

`cmd/scaffold.go` に以下のフラグを追加する:

```go
scaffoldFlagValues  []string // --v key=value (複数指定可)
scaffoldFlagDefault bool     // --default
```

### 5. プロンプトUIの改善

`gh pr create` 風のインタラクティブプロンプトを実装する。外部ライブラリ（survey/bubbletea等）は使用せず、ANSIエスケープコードを利用したカスタム実装とする（プロジェクトの依存関係を最小限に保つため）。

プロンプト表示例:
```
? Feature name (feature_name) (myprog): █
? Go module base name (go_module) (github.com/axsh/tokotachi/features): █
```

## 検証シナリオ (Verification Scenarios)

### シナリオ1: 重複質問の解消

1. `tt scaffold feature axsh-go-standard --cwd` を実行
2. `feature_name` を問われるのは1回だけであること
3. `Proceed? [y/N]` で `y` を入力
4. 再度 `feature_name` を問われないこと
5. scaffold が正常に適用されること

### シナリオ2: 全オプションのインタラクティブ問い合わせ

1. `tt scaffold feature axsh-go-standard --cwd` を実行
2. `feature_name` （required: true, default: "myprog"）について問い合わせがあること
3. `go_module` （required: false, default: "github.com/axsh/tokotachi/features"）についても問い合わせがあること
4. 各プロンプトにデフォルト値が表示されていること
5. `feature_name` で空Enterした場合、"myprog" が適用されること
6. `go_module` で空Enterした場合、デフォルト値が適用されること

### シナリオ3: `--v` オプションによる直接指定

1. `tt scaffold feature axsh-go-standard --cwd --v feature_name=foobar` を実行
2. `feature_name` のインタラクティブ問い合わせがスキップされること
3. `go_module` は通常通りインタラクティブに問い合わせされること
4. 生成されたファイルの `feature_name` が "foobar" であること

### シナリオ4: `--default` オプションの動作

1. `tt scaffold feature axsh-go-standard --cwd --default` を実行
2. `feature_name` （required: true）は問い合わせがあること
3. `go_module` （required: false）は問い合わせがなく、デフォルト値が適用されること

### シナリオ5: `--v` と `--default` の組み合わせ

1. `tt scaffold feature axsh-go-standard --cwd --v feature_name=foobar --default` を実行
2. `feature_name` も `go_module` もインタラクティブ問い合わせがないこと
3. `feature_name` = "foobar"、`go_module` = デフォルト値で適用されること

### シナリオ6: `--yes` との組み合わせ

1. `tt scaffold feature axsh-go-standard --cwd --yes --v feature_name=myapp --default` を実行
2. `Proceed? [y/N]` の確認もなく、インタラクティブ問い合わせもなく、即座に実行されること

## テスト項目 (Testing for the Requirements)

### 単体テスト

`template_test.go` の既存テストを更新し、以下のテストケースを追加・修正する:

| テストケース | 検証内容 |
|---|---|
| `TestCollectOptionValues_AllProvided` | 既存テストを更新: `useDefaults` パラメータ追加 |
| `TestCollectOptionValues_DefaultApplied` | 改修: `required: false` でもインタラクティブ問い合わせされることの確認（`useDefaults: false` の場合） |
| `TestCollectOptionValues_UseDefaults` | 新規: `useDefaults: true` で `required: false` の項目がスキップされることの確認 |
| `TestCollectOptionValues_InteractiveWithDefault` | 新規: デフォルト値のある項目で空Enter入力した場合にデフォルト値が適用されることの確認 |
| `TestCollectOptionValues_RequiredWithDefault` | 新規: `required: true` でデフォルト値がある場合でもインタラクティブ問い合わせされ、空Enterでデフォルト適用される確認 |
| `TestCollectOptionValues_InteractiveInput` | 既存テストを更新 |
| `TestCollectOptionValues_RequiredMissing` | 既存テストを維持 |
| `TestCollectOptionValues_NoOptions` | 既存テストを更新 |
| `TestParseOptionOverrides` | 新規: `--v key=value` のパースロジックのテスト |

### ビルド検証

```bash
scripts/process/build.sh
```

### 統合テスト

```bash
scripts/process/integration_test.sh
```

> **注**: 統合テストはGitHub APIを使用するため、既存テスト (`tt_scaffold_test.go`) の `--yes` フラグ使用時の動作に影響がないことを確認する。
