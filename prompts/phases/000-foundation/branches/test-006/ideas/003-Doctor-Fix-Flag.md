# devctl doctor --fix: 自動修正機能

## 背景 (Background)

`devctl doctor` は現在、チェック結果の報告と修正ヒント（FixHint）の表示のみを行う。しかし、自動的に修正可能な問題（例: `.devrc.yaml` が存在しない → デフォルト設定で生成）については、`--fix` フラグで自動修正を実行できると利便性が高い。

## 要件 (Requirements)

### 必須要件

1. **`--fix` フラグの追加**
   - `devctl doctor --fix` で、修正可能な項目を自動的に修正する
   - 修正した内容を出力に明記する（何を修正したかがユーザーに伝わること）

2. **修正可能な項目**
   - `.devrc.yaml` が存在しない場合: デフォルト設定でファイルを生成
     ```yaml
     project_name: ""
     default_editor: cursor
     default_container_mode: docker-local
     ```
   - `work/` ディレクトリが存在しない場合: ディレクトリを作成
   - `scripts/` ディレクトリが存在しない場合: ディレクトリを作成

3. **修正後の再チェック**
   - 修正を適用した後、再度チェックを実行して結果を表示する
   - もしくは、修正した項目については PASS + "🔧 fixed" のメッセージを表示する

4. **修正不可能な項目**
   - 外部ツール不足（git, docker 等）: 修正不可（FixHint のみ表示）
   - 不正な YAML/JSON: 修正不可（内容が不明なため）
   - `.devrc.yaml` に不正な値（unknown editor 等）: 修正不可

5. **出力フォーマット**
   - テキスト出力: 修正した項目に 🔧 アイコンを表示
     ```
     📋 Global Config (.devrc.yaml)
       🔧 .devrc.yaml          created with default settings
     ```
   - JSON出力: `"fixed": true` フィールドを追加
   - `--fix` フラグなし時は現在の動作と完全に同じ

## 実現方針 (Implementation Approach)

### データ構造の変更

`CheckResult` に `Fixed bool` フィールドを追加:

```go
type CheckResult struct {
    Category string `json:"category"`
    Name     string `json:"name"`
    Status   Status `json:"status"`
    Message  string `json:"message"`
    Expected string `json:"expected,omitempty"`
    FixHint  string `json:"fix_hint,omitempty"`
    Fixed    bool   `json:"fixed,omitempty"`
}
```

### フィクサー関数

各チェック関数に対応するフィクサー関数を用意するのではなく、`Run` 関数に `Fix bool` オプションを追加し、チェック後に修正可能な項目を修正 → 再チェックするアプローチを取る。

具体的には:
1. チェックを実行
2. `Fix` が true の場合、WARN/FAIL 結果のうち修正可能なものを修正
3. 修正した項目の結果を `Status: StatusPass, Fixed: true` に置き換え

### 修正ロジック

```go
// Fixer functions
func fixDevrcYAML(repoRoot string) error {
    // Create .devrc.yaml with defaults
}

func fixDirectory(repoRoot, dirName string) error {
    // os.MkdirAll
}
```

### cmd/doctor.go の変更

`--fix` フラグを追加し、`Options.Fix` に渡す。

## 検証シナリオ (Verification Scenarios)

1. `.devrc.yaml` が存在しない状態で `devctl doctor --fix` を実行:
   - `.devrc.yaml` が生成される
   - 出力に 🔧 アイコンと "created with default settings" が表示される
   - 終了コード 0

2. `work/` がない状態で `devctl doctor --fix` を実行:
   - `work/` ディレクトリが作成される
   - 出力に 🔧 アイコンが表示される

3. `--fix` なしで実行した場合は従来と同じ動作（何も修正されない）

4. `devctl doctor --fix --json` で JSON 出力に `"fixed": true` が含まれる

## テスト項目 (Testing for the Requirements)

| 要件 | テスト | 検証コマンド |
|---|---|---|
| Fix オプション追加 | `TestRun_WithFix` (単体テスト) | `scripts/process/build.sh` |
| .devrc.yaml 生成 | `TestFixDevrcYAML` (単体テスト) | `scripts/process/build.sh` |
| ディレクトリ作成 | `TestFixDirectory` (単体テスト) | `scripts/process/build.sh` |
| Fixed フィールド表示 | `TestReport_PrintText_Fixed` (単体テスト) | `scripts/process/build.sh` |
| 統合テスト | `TestDevctlDoctorFix` | `scripts/process/integration_test.sh --categories "integration-test" --specify "TestDevctlDoctorFix"` |
